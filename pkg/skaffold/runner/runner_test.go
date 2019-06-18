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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/client-go/tools/clientcmd/api"
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

	currentActions Actions
	actions        []Actions
	tag            int
}

func (t *TestBench) Labels() map[string]string                        { return map[string]string{} }
func (t *TestBench) TestDependencies() ([]string, error)              { return nil, nil }
func (t *TestBench) Dependencies() ([]string, error)                  { return nil, nil }
func (t *TestBench) Cleanup(ctx context.Context, out io.Writer) error { return nil }
func (t *TestBench) Prune(ctx context.Context, out io.Writer) error   { return nil }
func (t *TestBench) SyncMap(ctx context.Context, artifact *latest.Artifact) (map[string][]string, error) {
	return nil, nil
}
func (t *TestBench) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	return nil, nil
}

func (t *TestBench) enterNewCycle() {
	t.actions = append(t.actions, t.currentActions)
	t.currentActions = Actions{}
}

func (t *TestBench) Build(ctx context.Context, w io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
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

func (t *TestBench) Sync(ctx context.Context, item *sync.Item) error {
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

func (t *TestBench) Test(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
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

func (t *TestBench) Deploy(ctx context.Context, out io.Writer, artifacts []build.Artifact, labellers []deploy.Labeller) error {
	if len(t.deployErrors) > 0 {
		err := t.deployErrors[0]
		t.deployErrors = t.deployErrors[1:]
		if err != nil {
			return err
		}
	}

	t.currentActions.Deployed = findTags(artifacts)
	return nil
}

func (t *TestBench) Actions() []Actions {
	return append(t.actions, t.currentActions)
}

func findTags(artifacts []build.Artifact) []string {
	var tags []string
	for _, artifact := range artifacts {
		tags = append(tags, artifact.Tag)
	}
	return tags
}

func createRunner(t *testutil.T, testBench *TestBench) *SkaffoldRunner {
	opts := &config.SkaffoldOptions{
		Trigger: "polling",
	}

	cfg := &latest.SkaffoldConfig{}
	defaults.Set(cfg)

	runner, err := NewForConfig(opts, cfg)
	t.CheckNoError(err)

	runner.Builder = testBench
	runner.Syncer = testBench
	runner.Tester = testBench
	runner.Deployer = testBench

	return runner
}

func TestNewForConfig(t *testing.T) {
	var tests = []struct {
		description      string
		config           *latest.SkaffoldConfig
		shouldErr        bool
		cacheArtifacts   bool
		expectedBuilder  build.Builder
		expectedTester   test.Tester
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			config: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
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
			},
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "bad tagger config",
			config: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
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
			},
			shouldErr: true,
		},
		{
			description: "unknown builder",
			config: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{},
				},
			},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown tagger",
			config: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						TagPolicy: latest.TagPolicy{},
						BuildType: latest.BuildType{
							LocalBuild: &latest.LocalBuild{},
						},
					},
				}},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown deployer",
			config: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
						BuildType: latest.BuildType{
							LocalBuild: &latest.LocalBuild{},
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "no artifacts, cache",
			config: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
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
			},
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
			cacheArtifacts:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			cfg, err := NewForConfig(&config.SkaffoldOptions{
				Trigger: "polling",
			}, test.config)

			t.CheckError(test.shouldErr, err)
			if cfg != nil {
				b, _t, d := WithTimings(test.expectedBuilder, test.expectedTester, test.expectedDeployer, test.cacheArtifacts)

				t.CheckErrorAndTypeEquality(test.shouldErr, err, b, cfg.Builder)
				t.CheckErrorAndTypeEquality(test.shouldErr, err, _t, cfg.Tester)
				t.CheckErrorAndTypeEquality(test.shouldErr, err, d, cfg.Deployer)
			}
		})
	}
}
