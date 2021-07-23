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
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	kptV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/v2/kpt"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		tests := []struct {
			description string
			cfg         latestV2.DeployType
			helmVersion string
			expected    deploy.Deployer
			apply       bool
			shouldErr   bool
		}{
			{
				description: "no deployer",
				expected:    deploy.DeployerMux{},
			},
			{
				description: "helm deployer with 3.0.0 version",
				cfg:         latestV2.DeployType{HelmDeploy: &latestV2.HelmDeploy{}},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				expected:    deploy.NewDeployerMux([]deploy.Deployer{&helm.Deployer{}}, false),
			},
			{
				description: "helm deployer with less than 3.0.0 version",
				cfg:         latestV2.DeployType{HelmDeploy: &latestV2.HelmDeploy{}},
				helmVersion: "2.0.0",
				shouldErr:   true,
			},
			{
				description: "kubectl deployer",
				cfg:         latestV2.DeployType{KubectlDeploy: &latestV2.KubectlDeploy{}},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					t.RequireNonNilResult(kubectl.NewDeployer(&v2.RunContext{
						Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
					}, nil, deploy.NoopComponentProvider, &latestV2.KubectlDeploy{
						Flags: latestV2.KubectlFlags{},
					})).(deploy.Deployer),
				}, false),
			},
			{
				description: "kustomize deployer",
				cfg:         latestV2.DeployType{KustomizeDeploy: &latestV2.KustomizeDeploy{}},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					t.RequireNonNilResult(kustomize.NewDeployer(&v2.RunContext{
						Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
					}, nil, deploy.NoopComponentProvider, &latestV2.KustomizeDeploy{
						Flags: latestV2.KubectlFlags{},
					})).(deploy.Deployer),
				}, false),
			},
			{
				description: "kpt deployer",
				cfg:         latestV2.DeployType{KptDeploy: &latestV2.KptDeploy{}},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					&kpt.Deployer{},
				}, false),
			},
			{
				description: "kpt V2 deployer",
				cfg:         latestV2.DeployType{KptV2Deploy: &latestV2.KptV2Deploy{}},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					&kptV2.Deployer{},
				}, false),
			},
			{
				description: "apply forces creation of kubectl deployer with kpt config",
				cfg:         latestV2.DeployType{KptDeploy: &latestV2.KptDeploy{}},
				apply:       true,
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&v2.RunContext{
					Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
				}, nil, deploy.NoopComponentProvider, &latestV2.KubectlDeploy{
					Flags: latestV2.KubectlFlags{},
				})).(deploy.Deployer),
			},
			{
				description: "apply forces creation of kubectl deployer with helm config",
				cfg:         latestV2.DeployType{HelmDeploy: &latestV2.HelmDeploy{}},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				apply:       true,
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&v2.RunContext{
					Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
				}, nil, deploy.NoopComponentProvider, &latestV2.KubectlDeploy{
					Flags: latestV2.KubectlFlags{},
				})).(deploy.Deployer),
			},
			{
				description: "multiple deployers",
				cfg: latestV2.DeployType{
					HelmDeploy: &latestV2.HelmDeploy{},
					KptDeploy:  &latestV2.KptDeploy{},
				},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					&helm.Deployer{},
					&kpt.Deployer{},
				}, false),
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

				deployer, err := GetDeployer(&v2.RunContext{
					Opts: config.SkaffoldOptions{
						Apply: test.apply,
					},
					Pipelines: v2.NewPipelines([]latestV2.Pipeline{{
						Deploy: latestV2.DeployConfig{
							DeployType: test.cfg,
						},
					}}),
				}, deploy.NoopComponentProvider, nil)

				t.CheckError(test.shouldErr, err)
				t.CheckTypeEquality(test.expected, deployer)

				if reflect.TypeOf(test.expected) == reflect.TypeOf(deploy.DeployerMux{}) {
					expected := test.expected.(deploy.DeployerMux).GetDeployers()
					deployers := deployer.(deploy.DeployerMux).GetDeployers()
					t.CheckDeepEqual(len(expected), len(deployers))
					for i, v := range expected {
						t.CheckTypeEquality(v, deployers[i])
					}
				}
			})
		}
	})
}

func TestGetDefaultDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		tests := []struct {
			name      string
			cfgs      []latestV2.DeployType
			expected  *kubectl.Deployer
			shouldErr bool
		}{
			{
				name: "one config with kubectl deploy",
				cfgs: []latestV2.DeployType{{
					KubectlDeploy: &latestV2.KubectlDeploy{},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&v2.RunContext{
					Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
				}, nil, deploy.NoopComponentProvider, &latestV2.KubectlDeploy{
					Flags: latestV2.KubectlFlags{},
				})).(*kubectl.Deployer),
			},
			{
				name: "one config with kubectl deploy, with flags",
				cfgs: []latestV2.DeployType{{
					KubectlDeploy: &latestV2.KubectlDeploy{
						Flags: latestV2.KubectlFlags{
							Apply:  []string{"--foo"},
							Global: []string{"--bar"},
						},
					},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&v2.RunContext{
					Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
				}, nil, deploy.NoopComponentProvider, &latestV2.KubectlDeploy{
					Flags: latestV2.KubectlFlags{
						Apply:  []string{"--foo"},
						Global: []string{"--bar"},
					},
				})).(*kubectl.Deployer),
			},
			{
				name: "two kubectl configs with mismatched flags should fail",
				cfgs: []latestV2.DeployType{
					{
						KubectlDeploy: &latestV2.KubectlDeploy{
							Flags: latestV2.KubectlFlags{
								Apply: []string{"--foo"},
							},
						},
					},
					{
						KubectlDeploy: &latestV2.KubectlDeploy{
							Flags: latestV2.KubectlFlags{
								Apply: []string{"--bar"},
							},
						},
					},
				},
				shouldErr: true,
			},
			{
				name: "one config with helm deploy",
				cfgs: []latestV2.DeployType{{
					HelmDeploy: &latestV2.HelmDeploy{},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&v2.RunContext{
					Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
				}, nil, deploy.NoopComponentProvider, &latestV2.KubectlDeploy{
					Flags: latestV2.KubectlFlags{},
				})).(*kubectl.Deployer),
			},
			{
				name: "one config with kustomize deploy",
				cfgs: []latestV2.DeployType{{
					KustomizeDeploy: &latestV2.KustomizeDeploy{},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&v2.RunContext{
					Pipelines: v2.NewPipelines([]latestV2.Pipeline{{}}),
				}, nil, deploy.NoopComponentProvider, &latestV2.KubectlDeploy{
					Flags: latestV2.KubectlFlags{},
				})).(*kubectl.Deployer),
			},
		}

		for _, test := range tests {
			testutil.Run(tOuter, test.name, func(t *testutil.T) {
				pipelines := []latestV2.Pipeline{}
				for _, cfg := range test.cfgs {
					pipelines = append(pipelines, latestV2.Pipeline{
						Deploy: latestV2.DeployConfig{
							DeployType: cfg,
						},
					})
				}
				deployer, err := getDefaultDeployer(&v2.RunContext{
					Pipelines: v2.NewPipelines(pipelines),
				}, deploy.NoopComponentProvider, nil)

				t.CheckErrorAndFailNow(test.shouldErr, err)

				// if we were expecting an error, this implies that the returned deployer is nil
				// this error was checked in the previous call, so if we didn't fail there (i.e. the encountered error was correct),
				// then the test is finished and we can continue.
				if !test.shouldErr {
					t.CheckTypeEquality(&kubectl.Deployer{}, deployer)

					kDeployer := deployer.(*kubectl.Deployer)
					if !reflect.DeepEqual(kDeployer, test.expected) {
						t.Fail()
					}
				}
			})
		}
	})
}
