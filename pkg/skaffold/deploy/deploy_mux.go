/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deploy

import (
	"context"
	"io"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringset"
)

// DeployerMux forwards all method calls to the deployers it contains.
// When encountering an error, it aborts and returns the error. Otherwise,
// it collects the results and returns it in bulk.
type DeployerMux struct {
	iterativeStatusCheck bool
	deployers            []Deployer
}

type deployerWithHooks interface {
	HasRunnableHooks() bool
	PreDeployHooks(context.Context, io.Writer) error
	PostDeployHooks(context.Context, io.Writer) error
}

func NewDeployerMux(deployers []Deployer, iterativeStatusCheck bool) Deployer {
	return DeployerMux{deployers: deployers, iterativeStatusCheck: iterativeStatusCheck}
}

func (m DeployerMux) GetDeployers() []Deployer {
	return m.deployers
}

func (m DeployerMux) GetDeployersInverse() []Deployer {
	inverse := m.deployers
	for i, j := 0, len(inverse)-1; i < j; i, j = i+1, j-1 {
		inverse[i], inverse[j] = inverse[j], inverse[i]
	}
	return inverse
}

func (m DeployerMux) GetAccessor() access.Accessor {
	var accessors access.AccessorMux
	for _, deployer := range m.deployers {
		accessors = append(accessors, deployer.GetAccessor())
	}
	return accessors
}

func (m DeployerMux) GetDebugger() debug.Debugger {
	var debuggers debug.DebuggerMux
	for _, deployer := range m.deployers {
		debuggers = append(debuggers, deployer.GetDebugger())
	}
	return debuggers
}

func (m DeployerMux) GetLogger() log.Logger {
	var loggers log.LoggerMux
	for _, deployer := range m.deployers {
		loggers = append(loggers, deployer.GetLogger())
	}
	return loggers
}

func (m DeployerMux) GetStatusMonitor() status.Monitor {
	var monitors status.MonitorMux
	for _, deployer := range m.deployers {
		monitors = append(monitors, deployer.GetStatusMonitor())
	}
	return monitors
}

func (m DeployerMux) GetSyncer() sync.Syncer {
	var syncers sync.SyncerMux
	for _, deployer := range m.deployers {
		syncers = append(syncers, deployer.GetSyncer())
	}
	return syncers
}

func (m DeployerMux) RegisterLocalImages(images []graph.Artifact) {
	for _, deployer := range m.deployers {
		deployer.RegisterLocalImages(images)
	}
}

func (m DeployerMux) ConfigName() string {
	return ""
}

func (m DeployerMux) Deploy(ctx context.Context, w io.Writer, as []graph.Artifact, l manifest.ManifestListByConfig) error {
	for i, deployer := range m.deployers {
		eventV2.DeployInProgress(i)
		w, ctx = output.WithEventContext(ctx, w, constants.Deploy, strconv.Itoa(i))
		ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy")
		runHooks := false
		deployHooks, ok := deployer.(deployerWithHooks)
		if ok {
			runHooks = deployHooks.HasRunnableHooks()
		}
		if runHooks {
			if err := deployHooks.PreDeployHooks(ctx, w); err != nil {
				return err
			}
		}
		if err := deployer.Deploy(ctx, w, as, l); err != nil {
			eventV2.DeployFailed(i, err)
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
		// Always run iterative status check if there are deploy hooks.
		// This is required otherwise the deploy hooks can get erreneously executed on older pods from a previous deployment.
		if runHooks || m.iterativeStatusCheck {
			if err := deployer.GetStatusMonitor().Check(ctx, w); err != nil {
				eventV2.DeployFailed(i, err)
				endTrace(instrumentation.TraceEndError(err))
				return err
			}
		}
		if runHooks {
			if err := deployHooks.PostDeployHooks(ctx, w); err != nil {
				return err
			}
		}
		eventV2.DeploySucceeded(i)
		endTrace()
	}

	return nil
}

func (m DeployerMux) Dependencies() ([]string, error) {
	deps := stringset.New()
	for _, deployer := range m.deployers {
		result, err := deployer.Dependencies()
		if err != nil {
			return nil, err
		}
		deps.Insert(result...)
	}
	return deps.ToList(), nil
}

func (m DeployerMux) Cleanup(ctx context.Context, w io.Writer, dryRun bool, manifestsByConfig manifest.ManifestListByConfig) error {
	// Reverse order of deployers for cleanup to ensure resources
	// are removed before their definitions are removed.
	for _, deployer := range m.GetDeployersInverse() {
		ctx, endTrace := instrumentation.StartTrace(ctx, "Cleanup")
		if dryRun {
			output.Yellow.Fprintln(w, "Following resources would be deleted:")
		}
		if err := deployer.Cleanup(ctx, w, dryRun, manifestsByConfig); err != nil {
			output.Yellow.Fprintln(w, "Cleaning up resources encountered an error, will continue to clean up other resources.")
		}
		endTrace()
	}
	return nil
}

// TrackBuildArtifacts should *only* be called on individual deployers. This is a noop.
func (m DeployerMux) TrackBuildArtifacts(_, _ []graph.Artifact) {}
