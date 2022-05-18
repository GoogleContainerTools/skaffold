/*
Copyright 2022 The Skaffold Authors

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

package helm

import (
	"context"
	"fmt"
	"io"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Deployer31 deploys workflows using the helm CLI 3.1.0 or higher
type Deployer31 struct {
	*Deployer3
}

func NewDeployer31(ctx context.Context, cfg Config, labeller *label.DefaultLabeller, h *latest.HelmDeploy, hv semver.Version) (*Deployer31, error) {
	d3, err := NewBase(ctx, cfg, labeller, h, hv)
	if err != nil {
		return nil, err
	}
	return &Deployer31{
		Deployer3: d3,
	}, nil
}

// Deploy should ensure that the build results are deployed to the Kubernetes
// cluster.
func (h *Deployer31) Deploy(ctx context.Context, io io.Writer, graph []graph.Artifact) error {
	ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy", map[string]string{
		"DeployerType": "helm31",
	})
	defer endTrace()

	return fmt.Errorf("not yet implemented")
}

func (h *Deployer31) generateSkaffoldDebugFilter(buildsFile string) []string {
	args := []string{"filter", "--debugging", "--kube-context", h.kubeContext}
	if len(buildsFile) > 0 {
		args = append(args, "--build-artifacts", buildsFile)
	}
	args = append(args, h.Flags.Global...)

	if h.kubeConfig != "" {
		args = append(args, "--kubeconfig", h.kubeConfig)
	}
	return args
}
