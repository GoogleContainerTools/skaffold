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

package v2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/blang/semver"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type Actions struct {
	Built    []string
	Synced   []string
	Tested   []string
	Rendered []string
	Deployed []string
}

type TestBench struct {
	buildErrors   []error
	syncErrors    []error
	testErrors    []error
	renderErrors  []error
	deployErrors  []error
	namespaces    []string
	userIntents   []func(*runner.Intents)
	intents       *runner.Intents
	intentTrigger bool

	devLoop        func(context.Context, io.Writer, func() error) error
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

func (t *TestBench) WithRenderErrors(renderErrors []error) *TestBench {
	t.renderErrors = renderErrors
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

func (t *TestBench) GetAccessor() access.Accessor {
	return &access.NoopAccessor{}
}

func (t *TestBench) GetDebugger() debug.Debugger {
	return &debug.NoopDebugger{}
}

func (t *TestBench) GetLogger() log.Logger {
	return &log.NoopLogger{}
}

func (t *TestBench) GetStatusMonitor() status.Monitor {
	return &status.NoopMonitor{}
}

func (t *TestBench) GetSyncer() sync.Syncer {
	return t
}

func (t *TestBench) TrackBuildArtifacts(_ []graph.Artifact) {}

func (t *TestBench) TestDependencies(*latestV2.Artifact) ([]string, error) { return nil, nil }
func (t *TestBench) Dependencies() ([]string, error)                       { return nil, nil }
func (t *TestBench) Cleanup(context.Context, io.Writer) error              { return nil }
func (t *TestBench) Prune(context.Context, io.Writer) error                { return nil }

func (t *TestBench) enterNewCycle() {
	t.actions = append(t.actions, t.currentActions)
	t.currentActions = Actions{}
}

func (t *TestBench) Build(_ context.Context, _ io.Writer, _ tag.ImageTags, artifacts []*latestV2.Artifact) ([]graph.Artifact, error) {
	if len(t.buildErrors) > 0 {
		err := t.buildErrors[0]
		t.buildErrors = t.buildErrors[1:]
		if err != nil {
			return nil, err
		}
	}

	t.tag++

	var builds []graph.Artifact
	for _, artifact := range artifacts {
		builds = append(builds, graph.Artifact{
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

func (t *TestBench) Test(_ context.Context, _ io.Writer, artifacts []graph.Artifact) error {
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

func (t *TestBench) Deploy(_ context.Context, _ io.Writer, artifacts []graph.Artifact) ([]string, error) {
	if len(t.deployErrors) > 0 {
		err := t.deployErrors[0]
		t.deployErrors = t.deployErrors[1:]
		if err != nil {
			return nil, err
		}
	}

	t.currentActions.Deployed = findTags(artifacts)
	return t.namespaces, nil
}

func (t *TestBench) Render(_ context.Context, _ io.Writer, artifacts []graph.Artifact, _ bool, _ string) error {
	if len(t.renderErrors) > 0 {
		err := t.renderErrors[0]
		t.renderErrors = t.renderErrors[1:]
		if err != nil {
			return err
		}
	}
	t.currentActions.Rendered = findTags(artifacts)
	return nil
}

func (t *TestBench) Actions() []Actions {
	return append(t.actions, t.currentActions)
}

func (t *TestBench) WatchForChanges(ctx context.Context, out io.Writer, doDev func() error) error {
	// don't actually call the monitor here, because extra actions would be added
	if err := t.firstMonitor(true); err != nil {
		return err
	}

	t.intentTrigger = true
	for _, intent := range t.userIntents {
		intent(t.intents)
		if err := t.devLoop(ctx, out, doDev); err != nil {
			return err
		}
	}

	t.intentTrigger = false
	for i := 0; i < t.cycles; i++ {
		t.enterNewCycle()
		t.currentCycle = i
		if err := t.devLoop(ctx, out, doDev); err != nil {
			return err
		}
	}
	return nil
}

func (t *TestBench) LogWatchToUser(_ io.Writer) {}

func findTags(artifacts []graph.Artifact) []string {
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

type triggerState struct {
	build  bool
	sync   bool
	deploy bool
}

func createRunner(t *testutil.T, testBench *TestBench, monitor filemon.Monitor, artifacts []*latestV2.Artifact, autoTriggers *triggerState) *SkaffoldRunner {
	tmpDir := t.NewTempDir()
	tmpDir.Chdir()
	if autoTriggers == nil {
		autoTriggers = &triggerState{true, true, true}
	}
	cfg := &latestV2.SkaffoldConfig{
		Pipeline: latestV2.Pipeline{
			Build: latestV2.BuildConfig{
				TagPolicy: latestV2.TagPolicy{
					// Use the fastest tagger
					ShaTagger: &latestV2.ShaTagger{},
				},
				Artifacts: artifacts,
			},
			Deploy: latestV2.DeployConfig{StatusCheckDeadlineSeconds: 60},
		},
	}
	_ = defaults.Set(cfg)

	runCtx := &v2.RunContext{
		Pipelines: v2.NewPipelines([]latestV2.Pipeline{cfg.Pipeline}),
		Opts: config.SkaffoldOptions{
			Trigger:           "polling",
			WatchPollInterval: 100,
			AutoBuild:         autoTriggers.build,
			AutoSync:          autoTriggers.sync,
			AutoDeploy:        autoTriggers.deploy,
		},
		WorkingDir: tmpDir.Root(),
	}
	r, err := NewForConfig(runCtx)
	t.CheckNoError(err)

	r.Builder.Builder = testBench
	r.Tester = testBench
	r.renderer = testBench
	r.deployer = testBench
	r.listener = testBench
	r.monitor = monitor

	testBench.devLoop = func(ctx context.Context, out io.Writer, doDev func() error) error {
		if err := monitor.Run(true); err != nil {
			return err
		}
		return doDev()
	}

	testBench.firstMonitor = func(bool) error {
		// default to noop so we don't add extra actions
		return nil
	}

	return r
}

func TestNewForConfig(t *testing.T) {
	tests := []struct {
		description      string
		pipeline         latestV2.Pipeline
		shouldErr        bool
		cacheArtifacts   bool
		expectedBuilder  build.BuilderMux
		expectedTester   test.Tester
		expectedRenderer renderer.Renderer
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}},
					BuildType: latestV2.BuildType{
						LocalBuild: &latestV2.LocalBuild{},
					},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedRenderer: &renderer.SkaffoldRenderer{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "gcb config",
			pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}},
					BuildType: latestV2.BuildType{
						GoogleCloudBuild: &latestV2.GoogleCloudBuild{},
					},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedRenderer: &renderer.SkaffoldRenderer{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "cluster builder config",
			pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}},
					BuildType: latestV2.BuildType{
						Cluster: &latestV2.ClusterDetails{Timeout: "100s"},
					},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedRenderer: &renderer.SkaffoldRenderer{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "bad tagger config",
			pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{},
					BuildType: latestV2.BuildType{
						LocalBuild: &latestV2.LocalBuild{},
					},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{},
					},
				},
			},
			shouldErr: true,
		},
		{
			description:      "unknown builder and tagger",
			pipeline:         latestV2.Pipeline{},
			shouldErr:        true,
			expectedTester:   &test.FullTester{},
			expectedRenderer: &renderer.SkaffoldRenderer{},
			expectedDeployer: &kubectl.Deployer{},
		},
		{
			description: "no artifacts, cache",
			pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}},
					BuildType: latestV2.BuildType{
						LocalBuild: &latestV2.LocalBuild{},
					},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{},
					},
				},
			},
			expectedTester:   &test.FullTester{},
			expectedRenderer: &renderer.SkaffoldRenderer{},
			expectedDeployer: &kubectl.Deployer{},
			cacheArtifacts:   true,
		},
		{
			description: "raw renderer",
			pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}},
					BuildType: latestV2.BuildType{
						LocalBuild: &latestV2.LocalBuild{},
					},
				},
				Render: latestV2.RenderConfig{
					Generate: latestV2.Generate{RawK8s: []string{""}},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{},
					},
				},
			},
			expectedTester: &test.FullTester{},
			expectedRenderer: &renderer.SkaffoldRenderer{
				Generator: generate.Generator{},
			},
			expectedDeployer: &kubectl.Deployer{},
			cacheArtifacts:   true,
		},
		{
			description: "multiple deployers",
			pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}},
					BuildType: latestV2.BuildType{
						LocalBuild: &latestV2.LocalBuild{},
					},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy:   &latestV2.KubectlDeploy{},
						KustomizeDeploy: &latestV2.KustomizeDeploy{},
						HelmDeploy:      &latestV2.HelmDeploy{},
					},
				},
			},
			expectedTester: &test.FullTester{},
			expectedDeployer: deploy.NewDeployerMux([]deploy.Deployer{
				&helm.Deployer{},
				&kubectl.Deployer{},
				&kustomize.Deployer{},
			}, false),
		},
	}
	for _, tt := range tests {
		testutil.Run(t, tt.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&cluster.FindMinikubeBinary, func() (string, semver.Version, error) { return "", semver.Version{}, errors.New("not found") })
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput(
				"helm version --client", `version.BuildInfo{Version:"v3.0.0"}`).
				AndRunWithOutput("kubectl version --client -ojson", "v1.5.6"))
			tmpDir := t.NewTempDir()
			tmpDir.Chdir()
			runCtx := &v2.RunContext{
				Pipelines: v2.NewPipelines([]latestV2.Pipeline{tt.pipeline}),
				Opts: config.SkaffoldOptions{
					Trigger: "polling",
				},
				WorkingDir: tmpDir.Root(),
			}

			cfg, err := NewForConfig(runCtx)
			t.CheckError(tt.shouldErr, err)
			if cfg != nil {
				b, _t, r, d := runner.WithTimings(&tt.expectedBuilder, tt.expectedTester, tt.expectedRenderer,
					tt.expectedDeployer, tt.cacheArtifacts)
				if tt.shouldErr {
					t.CheckError(true, err)
				} else {
					t.CheckNoError(err)
					t.CheckTypeEquality(b, cfg.Pruner.Builder)
					t.CheckTypeEquality(_t, cfg.Tester)
					t.CheckTypeEquality(r, cfg.renderer)
					t.CheckTypeEquality(d, cfg.deployer)
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

	for _, tt := range tests {
		testutil.Run(t, tt.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			tmpDir.Chdir()
			opts := config.SkaffoldOptions{
				Trigger:           "polling",
				WatchPollInterval: 100,
				AutoBuild:         tt.autoBuild,
				AutoSync:          tt.autoSync,
				AutoDeploy:        tt.autoDeploy,
			}
			pipeline := latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					TagPolicy: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}},
					BuildType: latestV2.BuildType{
						LocalBuild: &latestV2.LocalBuild{},
					},
				},
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{},
					},
				},
			}
			r, _ := NewForConfig(&v2.RunContext{
				Opts:       opts,
				Pipelines:  v2.NewPipelines([]latestV2.Pipeline{pipeline}),
				WorkingDir: tmpDir.Root(),
			})

			r.intents.ResetBuild()
			r.intents.ResetSync()
			r.intents.ResetDeploy()

			b, s, d := r.intents.GetIntentsAttrs()
			t.CheckDeepEqual(tt.expectedBuildIntent, b)
			t.CheckDeepEqual(tt.expectedSyncIntent, s)
			t.CheckDeepEqual(tt.expectedDeployIntent, d)
		})
	}
}
