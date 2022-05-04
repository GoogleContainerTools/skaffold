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

package verify

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringset"
)

// VerifierMux forwards all method calls to the verifiers it contains.
// When encountering an error, it aborts and returns the error. Otherwise,
// it collects the results and returns it in bulk.
type VerifierMux struct {
	iterativeStatusCheck bool
	verifiers            []Verifier
}

type verifierWithHooks interface {
	HasRunnableHooks() bool
	PreDeployHooks(context.Context, io.Writer) error
	PostDeployHooks(context.Context, io.Writer) error
}

func NewVerifierMux(verifiers []Verifier, iterativeStatusCheck bool) Verifier {
	return VerifierMux{verifiers: verifiers, iterativeStatusCheck: iterativeStatusCheck}
}

func (m VerifierMux) GetVerifiers() []Verifier {
	return m.verifiers
}

func (m VerifierMux) GetAccessor() access.Accessor {
	var accessors access.AccessorMux
	for _, deployer := range m.verifiers {
		accessors = append(accessors, deployer.GetAccessor())
	}
	return accessors
}

func (m VerifierMux) GetLogger() log.Logger {
	var loggers log.LoggerMux
	for _, deployer := range m.verifiers {
		loggers = append(loggers, deployer.GetLogger())
	}
	return loggers
}

func (m VerifierMux) GetStatusMonitor() status.Monitor {
	var monitors status.MonitorMux
	for _, deployer := range m.verifiers {
		monitors = append(monitors, deployer.GetStatusMonitor())
	}
	return monitors
}

func (m VerifierMux) GetSyncer() sync.Syncer {
	var syncers sync.SyncerMux
	for _, deployer := range m.verifiers {
		syncers = append(syncers, deployer.GetSyncer())
	}
	return syncers
}

func (m VerifierMux) RegisterLocalImages(images []graph.Artifact) {
	for _, deployer := range m.verifiers {
		deployer.RegisterLocalImages(images)
	}
}

func (m VerifierMux) Verify(ctx context.Context, w io.Writer, as []graph.Artifact) error {
	for _, deployer := range m.verifiers {
		ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy")
		runHooks := false
		deployHooks, ok := deployer.(verifierWithHooks)
		if ok {
			runHooks = deployHooks.HasRunnableHooks()
		}
		if runHooks {
			if err := deployHooks.PreDeployHooks(ctx, w); err != nil {
				return err
			}
		}
		if err := deployer.Verify(ctx, w, as); err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
		// Always run iterative status check if there are deploy hooks.
		// This is required otherwise the deploy hooks can get erreneously executed on older pods from a previous deployment.
		if runHooks || m.iterativeStatusCheck {
			if err := deployer.GetStatusMonitor().Check(ctx, w); err != nil {
				endTrace(instrumentation.TraceEndError(err))
				return err
			}
		}
		if runHooks {
			if err := deployHooks.PostDeployHooks(ctx, w); err != nil {
				return err
			}
		}
		endTrace()
	}

	return nil
}

func (m VerifierMux) Dependencies() ([]string, error) {
	deps := stringset.New()
	for _, deployer := range m.verifiers {
		result, err := deployer.Dependencies()
		if err != nil {
			return nil, err
		}
		deps.Insert(result...)
	}
	return deps.ToList(), nil
}

func (m VerifierMux) Cleanup(ctx context.Context, w io.Writer, dryRun bool) error {
	for _, deployer := range m.verifiers {
		ctx, endTrace := instrumentation.StartTrace(ctx, "Cleanup")
		if dryRun {
			output.Yellow.Fprintln(w, "Following resources would be deleted:")
		}
		if err := deployer.Cleanup(ctx, w, dryRun); err != nil {
			return err
		}
		endTrace()
	}
	return nil
}

// TrackBuildArtifacts should *only* be called on individual verifiers. This is a noop.
func (m VerifierMux) TrackBuildArtifacts(_ []graph.Artifact) {}
