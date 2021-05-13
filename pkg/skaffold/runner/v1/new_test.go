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

package v1

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/preview"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		tests := []struct {
			description string
			cfg         latestV1.DeployType
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
				cfg:         latestV1.DeployType{HelmDeploy: &latestV1.HelmDeploy{}},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				expected: deploy.DeployerMux{
					&helm.Deployer{},
				},
			},
			{
				description: "helm deployer with less than 3.0.0 version",
				cfg:         latestV1.DeployType{HelmDeploy: &latestV1.HelmDeploy{}},
				helmVersion: "2.0.0",
				shouldErr:   true,
			},
			{
				description: "kubectl deployer",
				cfg:         latestV1.DeployType{KubectlDeploy: &latestV1.KubectlDeploy{}},
				expected: deploy.DeployerMux{
					t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
						Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
					}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KubectlDeploy{
						Flags: latestV1.KubectlFlags{},
					})).(deploy.Deployer),
				},
			},
			{
				description: "kustomize deployer",
				cfg:         latestV1.DeployType{KustomizeDeploy: &latestV1.KustomizeDeploy{}},
				expected: deploy.DeployerMux{
					t.RequireNonNilResult(kustomize.NewDeployer(&runcontext.RunContext{
						Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
					}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KustomizeDeploy{
						Flags: latestV1.KubectlFlags{},
					})).(deploy.Deployer),
				},
			},
			{
				description: "kpt deployer",
				cfg:         latestV1.DeployType{KptDeploy: &latestV1.KptDeploy{}},
				expected: deploy.DeployerMux{
					kpt.NewDeployer(&runcontext.RunContext{}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KptDeploy{}),
				},
			},
			{
				description: "apply forces creation of kubectl deployer with kpt config",
				cfg:         latestV1.DeployType{KptDeploy: &latestV1.KptDeploy{}},
				apply:       true,
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KubectlDeploy{
					Flags: latestV1.KubectlFlags{},
				})).(deploy.Deployer),
			},
			{
				description: "apply forces creation of kubectl deployer with helm config",
				cfg:         latestV1.DeployType{HelmDeploy: &latestV1.HelmDeploy{}},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				apply:       true,
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KubectlDeploy{
					Flags: latestV1.KubectlFlags{},
				})).(deploy.Deployer),
			},
			{
				description: "multiple deployers",
				cfg: latestV1.DeployType{
					HelmDeploy: &latestV1.HelmDeploy{},
					KptDeploy:  &latestV1.KptDeploy{},
				},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				expected: deploy.DeployerMux{
					&helm.Deployer{},
					kpt.NewDeployer(&runcontext.RunContext{}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KptDeploy{}),
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
					Opts: config.SkaffoldOptions{
						Apply: test.apply,
					},
					Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{
						Deploy: latestV1.DeployConfig{
							DeployType: test.cfg,
						},
					}}),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{})

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

func TestGetDefaultDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		tests := []struct {
			name      string
			cfgs      []latestV1.DeployType
			expected  *kubectl.Deployer
			shouldErr bool
		}{
			{
				name: "one config with kubectl deploy",
				cfgs: []latestV1.DeployType{{
					KubectlDeploy: &latestV1.KubectlDeploy{},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KubectlDeploy{
					Flags: latestV1.KubectlFlags{},
				})).(*kubectl.Deployer),
			},
			{
				name: "one config with kubectl deploy, with flags",
				cfgs: []latestV1.DeployType{{
					KubectlDeploy: &latestV1.KubectlDeploy{
						Flags: latestV1.KubectlFlags{
							Apply:  []string{"--foo"},
							Global: []string{"--bar"},
						},
					},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KubectlDeploy{
					Flags: latestV1.KubectlFlags{
						Apply:  []string{"--foo"},
						Global: []string{"--bar"},
					},
				})).(*kubectl.Deployer),
			},
			{
				name: "two kubectl configs with mismatched flags should fail",
				cfgs: []latestV1.DeployType{
					{
						KubectlDeploy: &latestV1.KubectlDeploy{
							Flags: latestV1.KubectlFlags{
								Apply: []string{"--foo"},
							},
						},
					},
					{
						KubectlDeploy: &latestV1.KubectlDeploy{
							Flags: latestV1.KubectlFlags{
								Apply: []string{"--bar"},
							},
						},
					},
				},
				shouldErr: true,
			},
			{
				name: "one config with helm deploy",
				cfgs: []latestV1.DeployType{{
					HelmDeploy: &latestV1.HelmDeploy{},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KubectlDeploy{
					Flags: latestV1.KubectlFlags{},
				})).(*kubectl.Deployer),
			},
			{
				name: "one config with kustomize deploy",
				cfgs: []latestV1.DeployType{{
					KustomizeDeploy: &latestV1.KustomizeDeploy{},
				}},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{}}),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{}, &latestV1.KubectlDeploy{
					Flags: latestV1.KubectlFlags{},
				})).(*kubectl.Deployer),
			},
		}

		for _, test := range tests {
			testutil.Run(tOuter, test.name, func(t *testutil.T) {
				pipelines := []latestV1.Pipeline{}
				for _, cfg := range test.cfgs {
					pipelines = append(pipelines, latestV1.Pipeline{
						Deploy: latestV1.DeployConfig{
							DeployType: cfg,
						},
					})
				}
				deployer, err := getDefaultDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(pipelines),
				}, nil, &log.NoopProvider{}, &preview.NoopPreviewer{})

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

func TestIsImageLocal(t *testing.T) {
	tests := []struct {
		description       string
		pushImagesFlagVal *bool
		localBuildConfig  *bool
		expected          bool
	}{
		{
			description:       "skaffold build --push=nil, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: nil,
			localBuildConfig:  nil,
			expected:          false,
		},
		{
			description:       "skaffold build --push=nil, pipeline.Build.LocalBuild.Push=false",
			pushImagesFlagVal: nil,
			localBuildConfig:  util.BoolPtr(false),
			expected:          true,
		},
		{
			description:       "skaffold build --push=nil, pipeline.Build.LocalBuild.Push=true",
			pushImagesFlagVal: nil,
			localBuildConfig:  util.BoolPtr(true),
			expected:          false,
		},
		{
			description:       "skaffold build --push=false, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(false),
			localBuildConfig:  nil,
			expected:          true,
		},
		{
			description:       "skaffold build --push=false, pipeline.Build.LocalBuild.Push=false",
			pushImagesFlagVal: util.BoolPtr(false),
			localBuildConfig:  util.BoolPtr(false),
			expected:          true,
		},
		{
			description:       "skaffold build --push=false, pipeline.Build.LocalBuild.Push=true",
			pushImagesFlagVal: util.BoolPtr(false),
			localBuildConfig:  util.BoolPtr(true),
			expected:          true,
		},
		{
			description:       "skaffold build --push=true, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(true),
			localBuildConfig:  nil,
			expected:          false,
		},
		{
			description:       "skaffold build --push=true, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(true),
			localBuildConfig:  util.BoolPtr(false),
			expected:          false,
		},
		{
			description:       "skaffold build --push=true, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(true),
			localBuildConfig:  util.BoolPtr(true),
			expected:          false,
		},
	}
	imageName := "testImage"
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			rctx := &runcontext.RunContext{
				Cluster: config.Cluster{
					PushImages: true,
				},
				Opts: config.SkaffoldOptions{
					PushImages: config.NewBoolOrUndefined(test.pushImagesFlagVal),
				},
				Pipelines: runcontext.NewPipelines([]latestV1.Pipeline{{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{ImageName: imageName},
						},
						BuildType: latestV1.BuildType{
							LocalBuild: &latestV1.LocalBuild{
								Push: test.localBuildConfig,
							},
						},
					},
				}})}
			output, _ := isImageLocal(rctx, imageName)
			if output != test.expected {
				t.Errorf("isImageLocal output was %t, expected: %t", output, test.expected)
			}
		})
	}
}
