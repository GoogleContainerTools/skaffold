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
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// DeployerMux forwards all method calls to the deployers it contains.
// When encountering an error, it aborts and returns the error. Otherwise,
// it collects the results and returns it in bulk.
type DeployerMux struct {
	iterativeStatusCheck bool
	deployers            []Deployer
}

func NewDeployerMux(deployers []Deployer, iterativeStatusCheck bool) Deployer {
	return DeployerMux{deployers: deployers, iterativeStatusCheck: iterativeStatusCheck}
}

func (m DeployerMux) GetDeployers() []Deployer {
	return m.deployers
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

func (m DeployerMux) Deploy(ctx context.Context, w io.Writer, as []graph.Artifact) error {
	for i, deployer := range m.deployers {
		eventV2.DeployInProgress(i)
		w, _ = output.WithEventContext(w, constants.Deploy, strconv.Itoa(i))
		ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy")

		if err := deployer.Deploy(ctx, w, as); err != nil {
			eventV2.DeployFailed(i, err)
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
		if m.iterativeStatusCheck {
			if err := deployer.GetStatusMonitor().Check(ctx, w); err != nil {
				eventV2.DeployFailed(i, err)
				endTrace(instrumentation.TraceEndError(err))
				return err
			}
		}
		eventV2.DeploySucceeded(i)
		endTrace()
	}

	return nil
}

func (m DeployerMux) Dependencies() ([]string, error) {
	deps := util.NewStringSet()
	for _, deployer := range m.deployers {
		result, err := deployer.Dependencies()
		if err != nil {
			return nil, err
		}
		deps.Insert(result...)
	}
	return deps.ToList(), nil
}

func (m DeployerMux) Cleanup(ctx context.Context, w io.Writer) error {
	for _, deployer := range m.deployers {
		ctx, endTrace := instrumentation.StartTrace(ctx, "Cleanup")
		if err := deployer.Cleanup(ctx, w); err != nil {
			return err
		}
		endTrace()
	}
	return nil
}

func (m DeployerMux) Render(ctx context.Context, w io.Writer, as []graph.Artifact, offline bool, filepath string) error {
	resources, buf := []string{}, &bytes.Buffer{}
	for _, deployer := range m.deployers {
		ctx, endTrace := instrumentation.StartTrace(ctx, "Render")
		buf.Reset()
		if err := deployer.Render(ctx, buf, as, offline, "" /* never write to files */); err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
		resources = append(resources, buf.String())
		endTrace()
	}

	allResources := strings.Join(resources, "\n---\n")
	return manifest.Write(strings.TrimSpace(allResources), filepath, w)
}

// TrackBuildArtifacts should *only* be called on individual deployers. This is a noop.
func (m DeployerMux) TrackBuildArtifacts(_ []graph.Artifact) {}
