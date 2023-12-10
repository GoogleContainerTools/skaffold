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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/verify"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/verify/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/verify/k8sjob"
)

// GetVerifier creates a verifier from a given RunContext and deploy pipeline definitions.
func GetVerifier(ctx context.Context, runCtx *runcontext.RunContext, labeller *label.DefaultLabeller) (verify.Verifier, error) {
	var verifiers []verify.Verifier
	var err error
	kubernetesTestCases := []*latest.VerifyTestCase{}
	localTestCases := []*latest.VerifyTestCase{}

	for _, p := range runCtx.GetPipelines() {
		for _, tc := range p.Verify {
			if tc.ExecutionMode.KubernetesClusterExecutionMode != nil {
				kubernetesTestCases = append(kubernetesTestCases, tc)
				continue
			}

			if tc.ExecutionMode.LocalExecutionMode == nil {
				tc.ExecutionMode.LocalExecutionMode = &latest.LocalVerifier{}
			}

			localTestCases = append(localTestCases, tc)
		}
	}
	envMap := map[string]string{}
	if runCtx.Opts.VerifyEnvFile != "" {
		envMap, err = util.ParseEnvVariablesFromFile(runCtx.Opts.VerifyEnvFile)
		if err != nil {
			return nil, err
		}
	}

	if len(kubernetesTestCases) != 0 {
		nv, err := k8sjob.NewVerifier(ctx, runCtx, labeller, kubernetesTestCases, runCtx.Artifacts(), envMap, runCtx.GetNamespace())
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, nv)
	}
	if len(localTestCases) != 0 {
		nv, err := docker.NewVerifier(ctx, runCtx, labeller, localTestCases, runCtx.PortForwardResources(), runCtx.VerifyDockerNetwork(), envMap)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, nv)
	}
	return verify.NewVerifierMux(verifiers, runCtx.IterativeStatusCheck()), nil
}
