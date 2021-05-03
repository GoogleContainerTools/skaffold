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

package v1

import (
	"errors"
	"testing"

	"github.com/blang/semver"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewForConfig(t *testing.T) {
	tests := []struct {
		description      string
		pipeline         latest_v1.Pipeline
		shouldErr        bool
		cacheArtifacts   bool
		expectedBuilder  build.BuilderMux
		expectedTester   test.Tester
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					TagPolicy: latest_v1.TagPolicy{ShaTagger: &latest_v1.ShaTagger{}},
					BuildType: latest_v1.BuildType{
						LocalBuild: &latest_v1.LocalBuild{},
					},
				},
				Deploy: latest_v1.DeployConfig{
					DeployType: latest_v1.DeployType{
						KubectlDeploy: &latest_v1.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "gcb config",
			pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					TagPolicy: latest_v1.TagPolicy{ShaTagger: &latest_v1.ShaTagger{}},
					BuildType: latest_v1.BuildType{
						GoogleCloudBuild: &latest_v1.GoogleCloudBuild{},
					},
				},
				Deploy: latest_v1.DeployConfig{
					DeployType: latest_v1.DeployType{
						KubectlDeploy: &latest_v1.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "cluster builder config",
			pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					TagPolicy: latest_v1.TagPolicy{ShaTagger: &latest_v1.ShaTagger{}},
					BuildType: latest_v1.BuildType{
						Cluster: &latest_v1.ClusterDetails{Timeout: "100s"},
					},
				},
				Deploy: latest_v1.DeployConfig{
					DeployType: latest_v1.DeployType{
						KubectlDeploy: &latest_v1.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "bad tagger config",
			pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					TagPolicy: latest_v1.TagPolicy{},
					BuildType: latest_v1.BuildType{
						LocalBuild: &latest_v1.LocalBuild{},
					},
				},
				Deploy: latest_v1.DeployConfig{
					DeployType: latest_v1.DeployType{
						KubectlDeploy: &latest_v1.KubectlDeploy{},
					},
				},
			},
			shouldErr: true,
		},
		{
			description:      "unknown builder and tagger",
			pipeline:         latest_v1.Pipeline{},
			shouldErr:        true,
			expectedTester:   &test.FullTester{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "no artifacts, cache",
			pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					TagPolicy: latest_v1.TagPolicy{ShaTagger: &latest_v1.ShaTagger{}},
					BuildType: latest_v1.BuildType{
						LocalBuild: &latest_v1.LocalBuild{},
					},
				},
				Deploy: latest_v1.DeployConfig{
					DeployType: latest_v1.DeployType{
						KubectlDeploy: &latest_v1.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &kubectl.Deployer{},
			cacheArtifacts:   true,
		},
		{
			description: "multiple deployers",
			pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					TagPolicy: latest_v1.TagPolicy{ShaTagger: &latest_v1.ShaTagger{}},
					BuildType: latest_v1.BuildType{
						LocalBuild: &latest_v1.LocalBuild{},
					},
				},
				Deploy: latest_v1.DeployConfig{
					DeployType: latest_v1.DeployType{
						KubectlDeploy:   &latest_v1.KubectlDeploy{},
						KustomizeDeploy: &latest_v1.KustomizeDeploy{},
						HelmDeploy:      &latest_v1.HelmDeploy{},
					},
				},
			},
			expectedTester: &test.FullTester{},
			expectedDeployer: deploy.DeployerMux([]deploy.Deployer{
				&helm.Deployer{},
				&kubectl.Deployer{},
				&kustomize.Deployer{},
			}),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&cluster.FindMinikubeBinary, func() (string, semver.Version, error) { return "", semver.Version{}, errors.New("not found") })
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput(
				"helm version --client", `version.BuildInfo{Version:"v3.0.0"}`).
				AndRunWithOutput("kubectl version --client -ojson", "v1.5.6"))

			runCtx := &runcontext.RunContext{
				Pipelines: runcontext.NewPipelines([]latest_v1.Pipeline{test.pipeline}),
				Opts: config.SkaffoldOptions{
					Trigger: "polling",
				},
			}

			cfg, err := NewForConfig(runCtx)
			t.CheckError(test.shouldErr, err)
			v1cfg := cfg.(*SkaffoldRunner)
			if v1cfg != nil {
				b, _t, d := runner.WithTimings(&test.expectedBuilder, test.expectedTester, test.expectedDeployer, test.cacheArtifacts)

				if test.shouldErr {
					t.CheckError(true, err)
				} else {
					t.CheckNoError(err)
					t.CheckTypeEquality(b, v1cfg.Builder)
					t.CheckTypeEquality(_t, v1cfg.Tester)
					t.CheckTypeEquality(d, v1cfg.Deployer)
				}
			}
		})
	}
}

func TestTriggerCallbackAndIntents(t *testing.T) {
	var tests = []struct {
		description          string
		autoBuild            bool
		autoSync             bool
		autoDeploy           bool
		expectedBuildIntent  bool
		expectedSyncIntent   bool
		expectedDeployIntent bool
	}{
		{
			description:          "default",
			autoBuild:            true,
			autoSync:             true,
			autoDeploy:           true,
			expectedBuildIntent:  true,
			expectedSyncIntent:   true,
			expectedDeployIntent: true,
		},
		{
			description:          "build trigger in api mode",
			autoBuild:            false,
			autoSync:             true,
			autoDeploy:           true,
			expectedBuildIntent:  false,
			expectedSyncIntent:   true,
			expectedDeployIntent: true,
		},
		{
			description:          "deploy trigger in api mode",
			autoBuild:            true,
			autoSync:             true,
			autoDeploy:           false,
			expectedBuildIntent:  true,
			expectedSyncIntent:   true,
			expectedDeployIntent: false,
		},
		{
			description:          "sync trigger in api mode",
			autoBuild:            true,
			autoSync:             false,
			autoDeploy:           true,
			expectedBuildIntent:  true,
			expectedSyncIntent:   false,
			expectedDeployIntent: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			opts := config.SkaffoldOptions{
				Trigger:           "polling",
				WatchPollInterval: 100,
				AutoBuild:         test.autoBuild,
				AutoSync:          test.autoSync,
				AutoDeploy:        test.autoDeploy,
			}
			pipeline := latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					TagPolicy: latest_v1.TagPolicy{ShaTagger: &latest_v1.ShaTagger{}},
					BuildType: latest_v1.BuildType{
						LocalBuild: &latest_v1.LocalBuild{},
					},
				},
				Deploy: latest_v1.DeployConfig{
					DeployType: latest_v1.DeployType{
						KubectlDeploy: &latest_v1.KubectlDeploy{},
					},
				},
			}
			r, _ := NewForConfig(&runcontext.RunContext{
				Opts:      opts,
				Pipelines: runcontext.NewPipelines([]latest_v1.Pipeline{pipeline}),
			})
			cfg := r.(*SkaffoldRunner)
			cfg.intents.ResetBuild()
			cfg.intents.ResetSync()
			cfg.intents.ResetDeploy()
			build, sync, deploy := runner.GetIntentsAttrs(*cfg.intents)
			t.CheckDeepEqual(test.expectedBuildIntent, build)
			t.CheckDeepEqual(test.expectedSyncIntent, sync)
			t.CheckDeepEqual(test.expectedDeployIntent, deploy)
		})
	}
}
