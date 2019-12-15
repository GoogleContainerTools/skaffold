/*
Copyright 2019 The Skaffold Authors

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
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type Actions struct {
	Built    []string
	Synced   []string
	Tested   []string
	Deployed []string
}

type TestBench struct {
	buildErrors  []error
	syncErrors   []error
	testErrors   []error
	deployErrors []error
	namespaces   []string

	devLoop        func(context.Context, io.Writer) error
	firstMonitor   func(bool) error
	cycles         int
	currentCycle   int
	currentActions Actions
	actions        []Actions
	tag            int
}

func NewTestBench() *TestBench {
	return &TestBench{}
}

func (t *TestBench) WithBuildErrors(buildErrors []error) *TestBench {
	t.buildErrors = buildErrors
	return t
}

func (t *TestBench) WithSyncErrors(syncErrors []error) *TestBench {
	t.syncErrors = syncErrors
	return t
}

func (t *TestBench) WithDeployErrors(deployErrors []error) *TestBench {
	t.deployErrors = deployErrors
	return t
}

func (t *TestBench) WithDeployNamespaces(ns []string) *TestBench {
	t.namespaces = ns
	return t
}

func (t *TestBench) WithTestErrors(testErrors []error) *TestBench {
	t.testErrors = testErrors
	return t
}

func (t *TestBench) Labels() map[string]string                        { return map[string]string{} }
func (t *TestBench) TestDependencies() ([]string, error)              { return nil, nil }
func (t *TestBench) Dependencies() ([]string, error)                  { return nil, nil }
func (t *TestBench) Cleanup(ctx context.Context, out io.Writer) error { return nil }
func (t *TestBench) Prune(ctx context.Context, out io.Writer) error   { return nil }
func (t *TestBench) SyncMap(ctx context.Context, artifact *latest.Artifact) (map[string][]string, error) {
	return nil, nil
}

func (t *TestBench) enterNewCycle() {
	t.actions = append(t.actions, t.currentActions)
	t.currentActions = Actions{}
}

func (t *TestBench) Build(_ context.Context, _ io.Writer, _ tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if len(t.buildErrors) > 0 {
		err := t.buildErrors[0]
		t.buildErrors = t.buildErrors[1:]
		if err != nil {
			return nil, err
		}
	}

	t.tag++

	var builds []build.Artifact
	for _, artifact := range artifacts {
		builds = append(builds, build.Artifact{
			ImageName: artifact.ImageName,
			Tag:       fmt.Sprintf("%s:%d", artifact.ImageName, t.tag),
		})
	}

	t.currentActions.Built = findTags(builds)
	return builds, nil
}

func (t *TestBench) Sync(_ context.Context, item *sync.Item) error {
	if len(t.syncErrors) > 0 {
		err := t.syncErrors[0]
		t.syncErrors = t.syncErrors[1:]
		if err != nil {
			return err
		}
	}

	t.currentActions.Synced = []string{item.Image}
	return nil
}

func (t *TestBench) Test(_ context.Context, _ io.Writer, artifacts []build.Artifact) error {
	if len(t.testErrors) > 0 {
		err := t.testErrors[0]
		t.testErrors = t.testErrors[1:]
		if err != nil {
			return err
		}
	}

	t.currentActions.Tested = findTags(artifacts)
	return nil
}

func (t *TestBench) Deploy(_ context.Context, _ io.Writer, artifacts []build.Artifact, _ []deploy.Labeller) *deploy.Result {
	if len(t.deployErrors) > 0 {
		err := t.deployErrors[0]
		t.deployErrors = t.deployErrors[1:]
		if err != nil {
			return deploy.NewDeployErrorResult(err)
		}
	}

	t.currentActions.Deployed = findTags(artifacts)
	return deploy.NewDeploySuccessResult(t.namespaces)
}

func (t *TestBench) Render(_ context.Context, _ io.Writer, artifacts []build.Artifact, _ string) error {
	return nil
}

func (t *TestBench) Actions() []Actions {
	return append(t.actions, t.currentActions)
}

func (t *TestBench) WatchForChanges(context.Context, io.Writer, func(context.Context, io.Writer) error) error {
	// don't actually call the monitor here, because extra actions would be added
	if err := t.firstMonitor(true); err != nil {
		return err
	}
	for i := 0; i < t.cycles; i++ {
		t.enterNewCycle()
		t.currentCycle = i
		if err := t.devLoop(context.Background(), ioutil.Discard); err != nil {
			return err
		}
	}
	return nil
}

func (t *TestBench) LogWatchToUser(_ io.Writer) {}

func findTags(artifacts []build.Artifact) []string {
	var tags []string
	for _, artifact := range artifacts {
		tags = append(tags, artifact.Tag)
	}
	return tags
}

func (r *SkaffoldRunner) WithMonitor(m filemon.Monitor) *SkaffoldRunner {
	r.monitor = m
	return r
}

func createRunner(t *testutil.T, testBench *TestBench, monitor filemon.Monitor) *SkaffoldRunner {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				TagPolicy: latest.TagPolicy{
					// Use the fastest tagger
					ShaTagger: &latest.ShaTagger{},
				},
			},
			Deploy: latest.DeployConfig{StatusCheckDeadlineSeconds: 60},
		},
	}
	defaults.Set(cfg)

	runCtx := &runcontext.RunContext{
		Cfg: cfg.Pipeline,
		Opts: config.SkaffoldOptions{
			Trigger:           "polling",
			WatchPollInterval: 100,
			AutoBuild:         true,
			AutoSync:          true,
			AutoDeploy:        true,
		},
	}
	runner, err := NewForConfig(runCtx)
	t.CheckNoError(err)

	runner.builder = testBench
	runner.syncer = testBench
	runner.tester = testBench
	runner.deployer = testBench
	runner.listener = testBench
	runner.monitor = monitor

	testBench.devLoop = func(context.Context, io.Writer) error {
		if err := monitor.Run(true); err != nil {
			return err
		}
		return runner.doDev(context.Background(), ioutil.Discard)
	}

	testBench.firstMonitor = func(bool) error {
		// default to noop so we don't add extra actions
		return nil
	}

	return runner
}

func TestNewForConfig(t *testing.T) {
	tests := []struct {
		description      string
		pipeline         latest.Pipeline
		shouldErr        bool
		cacheArtifacts   bool
		expectedBuilder  build.Builder
		expectedTester   test.Tester
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{},
					},
				},
			},
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "bad tagger config",
			pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{},
					},
				},
			},
			shouldErr: true,
		},
		{
			description:      "unknown builder and tagger",
			pipeline:         latest.Pipeline{},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown deployer",
			pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{},
			},
			shouldErr: true,
		},
		{
			description: "no artifacts, cache",
			pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{},
					},
				},
			},
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
			cacheArtifacts:   true,
		},
		{
			description: "multiple deployers",
			pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy:   &latest.KubectlDeploy{},
						KustomizeDeploy: &latest.KustomizeDeploy{},
						HelmDeploy:      &latest.HelmDeploy{},
					},
				},
			},
			expectedBuilder: &local.Builder{},
			expectedTester:  &test.FullTester{},
			expectedDeployer: deploy.DeployerMux([]deploy.Deployer{
				&deploy.HelmDeployer{},
				&deploy.KubectlDeployer{},
				&deploy.KustomizeDeployer{},
			}),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			runCtx := &runcontext.RunContext{
				Cfg: test.pipeline,
				Opts: config.SkaffoldOptions{
					Trigger: "polling",
				},
			}

			cfg, err := NewForConfig(runCtx)

			t.CheckError(test.shouldErr, err)
			if cfg != nil {
				b, _t, d := WithTimings(test.expectedBuilder, test.expectedTester, test.expectedDeployer, test.cacheArtifacts)

				t.CheckErrorAndTypeEquality(test.shouldErr, err, b, cfg.builder)
				t.CheckErrorAndTypeEquality(test.shouldErr, err, _t, cfg.tester)
				t.CheckErrorAndTypeEquality(test.shouldErr, err, d, cfg.deployer)
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
			pipeline := latest.Pipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{},
					},
				},
			}
			r, _ := NewForConfig(&runcontext.RunContext{
				Opts: opts,
				Cfg:  pipeline,
			})

			r.intents.resetBuild()
			r.intents.resetSync()
			r.intents.resetDeploy()

			t.CheckDeepEqual(test.expectedBuildIntent, r.intents.build)
			t.CheckDeepEqual(test.expectedSyncIntent, r.intents.sync)
			t.CheckDeepEqual(test.expectedDeployIntent, r.intents.deploy)
		})
	}
}
