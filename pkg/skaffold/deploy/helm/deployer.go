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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// NewDeployer returns a configured Deployer3 if version is less than 3.1
// else returns Deployer31 with post-render functionality
func NewDeployer(ctx context.Context, cfg Config, labeller *label.DefaultLabeller, h *latest.HelmDeploy) (deploy.Deployer, error) {
	hv, err := binVer(ctx)
	if err != nil {
		return nil, versionGetErr(err)
	}

	if hv.LT(helm3Version) {
		return nil, minVersionErr()
	}
	if hv.LT(helm31Version) {
		return NewDeployer30(ctx, cfg, labeller, h, hv)
	}
	return NewDeployer31(ctx, cfg, labeller, h, hv)
}
