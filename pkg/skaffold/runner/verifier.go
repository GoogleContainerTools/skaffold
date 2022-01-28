/*
Copyright 2021 The Skaffold Authors

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

package runner

import (
	"context"
	"errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/verify"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/verify/docker"
)

// GetVerifier creates a verifier from a given RunContext and deploy pipeline definitions.
func GetVerifier(ctx context.Context, runCtx *runcontext.RunContext, labeller *label.DefaultLabeller) (verify.Deployer, error) {
	localDeploy := false
	remoteDeploy := false

	var deployers []verify.Deployer
	localDeploy = true

	// TODO(aaron-prindle) vvvvv see if below is still required ======
	// Override the cluster on the runcontext.
	// This is used to determine whether we should push images, and we want to avoid that unless explicitly asked for.
	// Safe to do because we explicitly disallow simultaneous remote and local deployments.
	runCtx.Cluster = config.Cluster{
		Local:      true,
		PushImages: false,
		LoadImages: false,
	}

	if localDeploy && remoteDeploy {
		return nil, errors.New("docker deployment not supported alongside cluster deployments")
	}

	// TODO(aaron-prindle) verify with Gaurav that runCtx.GetPipelines gets the correct Pipelines in this case
	for _, p := range runCtx.GetPipelines() {
		d, err := docker.NewVerifier(ctx, runCtx, labeller, p.Verify, runCtx.PortForwardResources(), runCtx.VerifyDockerNetwork())
		if err != nil {
			return nil, err
		}
		deployers = append(deployers, d)
	}

	return verify.NewDeployerMux(deployers, runCtx.IterativeStatusCheck()), nil
}
