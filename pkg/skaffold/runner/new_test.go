/*
Copyright 2020 The Skaffold Authors

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
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		tests := []struct {
			description string
			cfg         latest.DeployType
			helmVersion string
			expected    deploy.Deployer
			shouldErr   bool
		}{
			{
				description: "no deployer",
				expected:    deploy.DeployerMux{},
			},
			{
				description: "helm deployer with 3.0.0 version",
				cfg:         latest.DeployType{HelmDeploy: &latest.HelmDeploy{}},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				expected:    &helm.Deployer{},
			},
			{
				description: "helm deployer with less than 3.0.0 version",
				cfg:         latest.DeployType{HelmDeploy: &latest.HelmDeploy{}},
				helmVersion: "2.0.0",
				shouldErr:   true,
			},
			{
				description: "kubectl deployer",
				cfg:         latest.DeployType{KubectlDeploy: &latest.KubectlDeploy{}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latest.Pipeline{{}}),
				}, nil, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				})).(deploy.Deployer),
			},
			{
				description: "kustomize deployer",
				cfg:         latest.DeployType{KustomizeDeploy: &latest.KustomizeDeploy{}},
				expected: t.RequireNonNilResult(kustomize.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latest.Pipeline{{}}),
				}, nil, &latest.KustomizeDeploy{
					Flags: latest.KubectlFlags{},
				})).(deploy.Deployer),
			},
			{
				description: "kpt deployer",
				cfg:         latest.DeployType{KptDeploy: &latest.KptDeploy{}},
				expected:    kpt.NewDeployer(&runcontext.RunContext{}, nil, &latest.KptDeploy{}),
			},
			{
				description: "multiple deployers",
				cfg: latest.DeployType{
					HelmDeploy: &latest.HelmDeploy{},
					KptDeploy:  &latest.KptDeploy{},
				},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				expected: deploy.DeployerMux{
					&helm.Deployer{},
					kpt.NewDeployer(&runcontext.RunContext{}, nil, &latest.KptDeploy{}),
				},
			},
		}
		for _, test := range tests {
			testutil.Run(tOuter, test.description, func(t *testutil.T) {
				if test.helmVersion != "" {
					t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput(
						"helm version --client",
						test.helmVersion,
					))
				}

				deployer, err := getDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latest.Pipeline{{
						Deploy: latest.DeployConfig{
							DeployType: test.cfg,
						},
					}}),
				}, nil)

				t.CheckError(test.shouldErr, err)
				t.CheckTypeEquality(test.expected, deployer)

				if reflect.TypeOf(test.expected) == reflect.TypeOf(deploy.DeployerMux{}) {
					expected := test.expected.(deploy.DeployerMux)
					deployers := deployer.(deploy.DeployerMux)
					t.CheckDeepEqual(len(expected), len(deployers))
					for i, v := range expected {
						t.CheckTypeEquality(v, deployers[i])
					}
				}
			})
		}
	})
}
