/*
Copyright 2024 The Skaffold Authors

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

package tofu

import (
	"context"
	"fmt"
	"io"

	"github.com/segmentio/textio"
	"go.opentelemetry.io/otel/trace"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/sync"
)

// Deployer deploys Workspaces using tofu CLI.
type Deployer struct {
	configName string

	*latest.TofuDeploy

	tofu CLI
}

// NewDeployer returns a new Deployer for a DeployConfig filled
// with the needed configuration for `tofu apply`
func NewDeployer(cfg Config, labeller *label.DefaultLabeller, d *latest.TofuDeploy, artifacts []*latest.Artifact, configName string) (*Deployer, error) {

	if d.Workspace == "" {
		return nil, fmt.Errorf("tofu needs Workspace to deploy from: %s", d.Workspace)
	}

	tofu := NewCLI(cfg)

	return &Deployer{
		configName: configName,
		TofuDeploy: d,
		tofu:       tofu,
	}, nil
}

func (k *Deployer) ConfigName() string {
	return k.configName
}

// GetAccessor not supported
func (k *Deployer) GetAccessor() access.Accessor {
	return &access.NoopAccessor{}
}

// GetDebugger not supported.
func (k *Deployer) GetDebugger() debug.Debugger {
	return &debug.NoopDebugger{}
}

// GetLogger not supported.
func (k *Deployer) GetLogger() log.Logger {
	return &log.NoopLogger{}
}

// GetStatusMonitor not supported.
func (k *Deployer) GetStatusMonitor() status.Monitor {
	return &status.NoopMonitor{}
}

// GetSyncer not supported.
func (k *Deployer) GetSyncer() sync.Syncer {
	return &sync.NoopSyncer{}
}

// RegisterLocalImages not implemented
func (k *Deployer) RegisterLocalImages(images []graph.Artifact) {
}

// TrackBuildArtifacts not implemented
func (k *Deployer) TrackBuildArtifacts(builds, deployedImages []graph.Artifact) {
}

// Deploy runs `tofu apply` on Workspaces
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact, manifestsByConfig manifest.ManifestListByConfig) error {

	var (
		childCtx context.Context
		endTrace func(...trace.SpanEndOption)
	)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "tofu",
	})

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_TofuApply")
	if err := k.tofu.Apply(childCtx, textio.NewPrefixWriter(out, " - ")); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

// HasRunnableHooks not supported
func (k *Deployer) HasRunnableHooks() bool {
	return false
}

// PreDeployHooks not supported
func (k *Deployer) PreDeployHooks(ctx context.Context, out io.Writer) error {
	_, endTrace := instrumentation.StartTrace(ctx, "Deploy_PreHooks")
	endTrace()
	return nil
}

// PostDeployHooks not supported
func (k *Deployer) PostDeployHooks(ctx context.Context, out io.Writer) error {
	_, endTrace := instrumentation.StartTrace(ctx, "Deploy_PostHooks")
	endTrace()
	return nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool, manifestsByConfig manifest.ManifestListByConfig) error {

	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "tofu",
	})
	if dryRun {
		return nil
	}
	// TODO: Implement delete

	return nil
}

// Dependencies lists all the files that describe what needs to be deployed.
func (k *Deployer) Dependencies() ([]string, error) {
	return []string{}, nil
}
