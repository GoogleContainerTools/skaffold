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
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/cloudrun"
	component "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/component/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/helm"
	kptV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	k8sloader "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/loader"
	k8slogger "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/portforward"
	k8sstatus "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		tests := []struct {
			description       string
			cfg               latest.Pipeline
			helmVersion       string
			expected          deploy.Deployer
			apply             bool
			shouldErr         bool
			deepCheckDeployer bool
		}{
			{
				description: "no deployer",
				expected:    deploy.DeployerMux{},
			},
			{
				description: "helm deployer with 3.1.0 version",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{},
						},
					},
				},
				helmVersion: `version.BuildInfo{Version:"v3.1.0"}`,
				expected:    deploy.NewDeployerMux([]deploy.Deployer{&helm.Deployer{}}, false),
			},
			{
				description: "helm deployer with less than 3.0.0 version",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{},
						},
					},
				},
				helmVersion: "2.0.0",
				shouldErr:   true,
			},
			{
				description: "kubectl deployer",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{KubectlDeploy: &latest.KubectlDeploy{}},
					},
				},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
						Pipelines: runcontext.NewPipelines(
							map[string]latest.Pipeline{
								"default": {},
							},
							[]string{"default"}),
					}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
						Flags: latest.KubectlFlags{},
					}, nil, "default", nil)).(deploy.Deployer),
				}, false),
			},
			{
				description: "kpt deployer",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{KptDeploy: &latest.KptDeploy{}},
					},
				},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					&kptV2.Deployer{},
				}, false),
			},
			{
				description: "cloud run deployer",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							10*time.Minute,
							false,
							util.Ptr(true)),
						).(deploy.Deployer)},
					false),
			},
			{
				description: "cloud run deployer with StatusCheckDeadlineSeconds specified",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						StatusCheckDeadlineSeconds: 300,
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							5*time.Minute,
							false,
							util.Ptr(true)),
						).(deploy.Deployer)},
					false),
			},
			{
				description: "cloud run deployer with tolerateFailures set to true",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						StatusCheckDeadlineSeconds:    900,
						TolerateFailuresUntilDeadline: true,
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							15*time.Minute,
							true,
							util.Ptr(true)),
						).(deploy.Deployer)},
					false),
			},
			{
				description: "cloud run deployer with statusCheck disabled",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						StatusCheck: util.Ptr(false),
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							10*time.Minute,
							false,
							util.Ptr(false)),
						).(deploy.Deployer)},
					false),
			},
			{
				description: "apply forces creation of kubectl deployer with kpt config",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{KptDeploy: &latest.KptDeploy{}},
					},
				},
				apply: true,
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				}, nil, "default", nil)).(deploy.Deployer),
			},
			{
				description: "apply forces creation of kubectl deployer with helm config",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{},
						},
					},
				},
				helmVersion: `version.BuildInfo{Version:"v3.0.0"}`,
				apply:       true,
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				}, nil, "default", nil)).(deploy.Deployer),
			},
			{
				description: "multiple deployers",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy:    &latest.KubectlDeploy{},
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{},
						},
					},
				},
				helmVersion: `version.BuildInfo{Version:"v3.7.0"}`,
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					&helm.Deployer{},
					&kubectl.Deployer{},
				}, false),
			},
			{
				description: "multiple deployers with kpt",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KptDeploy:        &latest.KptDeploy{},
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{},
						},
					},
				},
				shouldErr:   true,
				helmVersion: `version.BuildInfo{Version:"v3.7.0"}`,
			},
			{
				description: "apply does not allow multiple deployers when a helm namespace is set",
				apply:       true,
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{
								Releases: []latest.HelmRelease{{Namespace: "foo"}},
							},
							KubectlDeploy: &latest.KubectlDeploy{},
						},
					},
				},
				shouldErr: true,
			},
			{
				description: "apply does not allow multiple helm releases with different namespaces set",
				apply:       true,
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{
								Releases: []latest.HelmRelease{
									{
										Namespace: "foo",
									},
									{
										Namespace: "bar",
									},
								},
							},
						},
					},
				},
				shouldErr: true,
			},
			{
				description: "apply does allow multiple helm releases with the same namespace set",
				apply:       true,
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{
								Releases: []latest.HelmRelease{
									{
										Namespace: "foo",
									},
									{
										Namespace: "foo",
									},
								},
							},
						},
					},
				},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				}, nil, "default", nil)).(deploy.Deployer),
			},
			{
				description: "apply works with Cloud Run",
				apply:       true,
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "TestProject", Region: "us-central1"},
						},
					},
				},
				expected: t.RequireNonNilResult(cloudrun.NewDeployer(
					&runcontext.RunContext{},
					&label.DefaultLabeller{},
					&latest.CloudRunDeploy{
						ProjectID: "TestProject",
						Region:    "us-central1",
					},
					"default",
					10*time.Minute,
					false,
					util.Ptr(true)),
				).(deploy.Deployer),
				deepCheckDeployer: true,
			},
			{
				description: "apply does not allow multiple deployers when Cloud Run is used",
				apply:       true,
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
							KubectlDeploy:  &latest.KubectlDeploy{},
						},
					},
				},
				shouldErr: true,
			},
		}
		for _, test := range tests {
			testutil.Run(tOuter, test.description, func(t *testutil.T) {
				if test.helmVersion != "" {
					t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput(
						"helm version",
						test.helmVersion,
					))
				}

				deployer, err := GetDeployer(context.Background(), &runcontext.RunContext{
					Opts: config.SkaffoldOptions{
						Apply: test.apply,
					},
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": test.cfg,
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, "", false)

				t.CheckError(test.shouldErr, err)
				t.CheckTypeEquality(test.expected, deployer)

				if reflect.TypeOf(test.expected) == reflect.TypeOf(deploy.DeployerMux{}) {
					expected := test.expected.(deploy.DeployerMux).GetDeployers()
					deployers := deployer.(deploy.DeployerMux).GetDeployers()
					t.CheckDeepEqual(len(expected), len(deployers))
					for i, v := range expected {
						t.CheckTypeEquality(v, deployers[i])
						if test.deepCheckDeployer {
							t.CheckDeepEqual(v, deployers[i], cmpopts.IgnoreUnexported(cloudrun.Deployer{}, helm.Deployer{}, kubectl.Deployer{}, kptV2.Deployer{}))
						}
					}
				} else if test.deepCheckDeployer {
					// it's not a mux so if we want to deep check, do so directly.
					t.CheckDeepEqual(test.expected, deployer, cmpopts.IgnoreUnexported(cloudrun.Deployer{}, helm.Deployer{}, kubectl.Deployer{}, kptV2.Deployer{}))
				}
			})
		}
	})
}

func TestGetDefaultDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		t.Override(&component.NewAccessor, func(portforward.Config, string, *pkgkubectl.CLI, kubernetes.PodSelector, label.Config, *[]string) access.Accessor {
			return &access.NoopAccessor{}
		})
		t.Override(&component.NewDebugger, func(config.RunMode, kubernetes.PodSelector, *[]string, string) debug.Debugger {
			return &debug.NoopDebugger{}
		})
		t.Override(&component.NewMonitor, func(k8sstatus.Config, string, *label.DefaultLabeller, *[]string, []manifest.GroupKindSelector) k8sstatus.Monitor {
			return &k8sstatus.NoopMonitor{}
		})
		t.Override(&component.NewImageLoader, func(k8sloader.Config, *pkgkubectl.CLI) loader.ImageLoader {
			return &loader.NoopImageLoader{}
		})
		t.Override(&component.NewSyncer, func(*pkgkubectl.CLI, *[]string, k8slogger.Formatter) sync.Syncer {
			return &sync.NoopSyncer{}
		})
		t.Override(&component.NewLogger, func(k8slogger.Config, *pkgkubectl.CLI, kubernetes.PodSelector, *[]string) k8slogger.Logger {
			return &k8slogger.NoopLogger{}
		})
		const configNameForDefaultDeployer = ""
		tests := []struct {
			name      string
			cfgs      map[string]latest.DeployType
			expected  *kubectl.Deployer
			shouldErr bool
		}{
			{
				name: "one config with kubectl deploy",
				cfgs: map[string]latest.DeployType{
					configNameForDefaultDeployer: {KubectlDeploy: &latest.KubectlDeploy{}},
				},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							configNameForDefaultDeployer: {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				}, nil, configNameForDefaultDeployer, nil)).(*kubectl.Deployer),
			},
			{
				name: "one config with kubectl deploy, with flags",
				cfgs: map[string]latest.DeployType{
					configNameForDefaultDeployer: {
						KubectlDeploy: &latest.KubectlDeploy{
							Flags: latest.KubectlFlags{
								Apply:  []string{"--foo"},
								Global: []string{"--bar"},
							},
						},
					},
				},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							configNameForDefaultDeployer: {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{
						Apply:  []string{"--foo"},
						Global: []string{"--bar"},
					},
				}, nil, configNameForDefaultDeployer, nil)).(*kubectl.Deployer),
			},
			{
				name: "two kubectl configs with mismatched flags should fail",
				cfgs: map[string]latest.DeployType{
					"config1": {
						KubectlDeploy: &latest.KubectlDeploy{
							Flags: latest.KubectlFlags{
								Apply: []string{"--foo"},
							},
						},
					},
					"config2": {
						KubectlDeploy: &latest.KubectlDeploy{
							Flags: latest.KubectlFlags{
								Apply: []string{"--bar"},
							},
						},
					},
				},
				shouldErr: true,
			},
			{
				name: "one config with helm deploy",
				cfgs: map[string]latest.DeployType{
					configNameForDefaultDeployer: {LegacyHelmDeploy: &latest.LegacyHelmDeploy{}},
				},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							configNameForDefaultDeployer: {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				}, nil, configNameForDefaultDeployer, nil)).(*kubectl.Deployer),
			},
		}

		for _, test := range tests {
			testutil.Run(tOuter, test.name, func(t *testutil.T) {
				pipelines := map[string]latest.Pipeline{}
				var orderedConfigNames []string
				for configName, cfg := range test.cfgs {
					pipeline := latest.Pipeline{
						Deploy: latest.DeployConfig{
							DeployType: cfg,
						},
					}
					pipelines[configName] = pipeline
					orderedConfigNames = append(orderedConfigNames, configName)
				}
				deployer, err := getDefaultDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(pipelines, orderedConfigNames),
				}, &label.DefaultLabeller{}, nil)

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

func TestGetCloudRunDeployer(tOuter *testing.T) {
	tests := []struct {
		name     string
		opts     config.SkaffoldOptions
		cfgs     map[string]latest.DeployType
		expected *cloudrun.Deployer
		haveErr  bool
	}{
		{
			name: "deploy one config with no flags set",
			cfgs: map[string]latest.DeployType{"": {
				CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"},
			}},
			expected: &cloudrun.Deployer{Project: "test-project", Region: "test-region", CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"}},
		},
		{
			name: "deploy with two configs and conflicting processes",
			cfgs: map[string]latest.DeployType{"": {
				CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"}},
				"second": {
					CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project2", Region: "test-region"},
				},
			},
			haveErr: true,
		},
		{
			name: "deploy with flags set overrides config",
			opts: config.SkaffoldOptions{CloudRunProject: "overridden-project"},
			cfgs: map[string]latest.DeployType{"": {
				CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"},
			}},
			expected: &cloudrun.Deployer{Project: "overridden-project", Region: "test-region", CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "overridden-project", Region: "test-region"}},
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.name, func(t *testutil.T) {
			pipelines := make(map[string]latest.Pipeline)
			var orderedConfigNames []string
			for name, config := range test.cfgs {
				pipelines[name] = latest.Pipeline{Deploy: latest.DeployConfig{DeployType: config}}
				orderedConfigNames = append(orderedConfigNames, name)
			}
			rctx := &runcontext.RunContext{
				Opts:      test.opts,
				Pipelines: runcontext.NewPipelines(pipelines, orderedConfigNames),
			}
			crDeployer, err := getCloudRunDeployer(rctx, &label.DefaultLabeller{}, rctx.DeployConfigs(), "")
			t.CheckErrorAndFailNow(test.haveErr, err)
			t.CheckDeepEqual(crDeployer, test.expected, cmpopts.IgnoreUnexported(cloudrun.Deployer{}))
		})
	}
}
