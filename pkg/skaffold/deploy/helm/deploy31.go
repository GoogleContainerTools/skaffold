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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/blang/semver"
)

// Deployer31 deploys workflows using the helm CLI 3.1.0 or higher
type Deployer31 struct {
	*Deployer3
}

func NewDeployer31(ctx context.Context, cfg Config, labeller *label.DefaultLabeller, h *latestV1.HelmDeploy, hv semver.Version) (*Deployer31, error) {
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
func (h *Deployer31) Deploy(context.Context, io.Writer, []graph.Artifact) error {

	return fmt.Errorf("not yet implemented")
}

// Render should ensure that the build results are deployed to the Kubernetes
// cluster.
func (h *Deployer31) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) error {

	return fmt.Errorf("not yet implemented")
}
