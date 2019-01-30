/*
Copyright 2018 The Skaffold Authors

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

	currentActions Actions
	actions        []Actions
	tag            int
}

func (t *TestBench) Labels() map[string]string                        { return map[string]string{} }
func (t *TestBench) TestDependencies() ([]string, error)              { return nil, nil }
func (t *TestBench) Dependencies() ([]string, error)                  { return nil, nil }
func (t *TestBench) Cleanup(ctx context.Context, out io.Writer) error { return nil }

func (t *TestBench) enterNewCycle() {
	t.actions = append(t.actions, t.currentActions)
	t.currentActions = Actions{}
}

func (t *TestBench) Build(ctx context.Context, w io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
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

	t.currentActions.Built = tags(builds)
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

	t.currentActions.Tested = tags(artifacts)
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

	t.currentActions.Deployed = tags(artifacts)
	return nil
}

func (t *TestBench) Actions() []Actions {
	return append(t.actions, t.currentActions)
}

func tags(artifacts []build.Artifact) []string {
	var tags []string
	for _, artifact := range artifacts {
		tags = append(tags, artifact.Tag)
	}
	return tags
}

func createRunner(t *testing.T, testBench *TestBench) *SkaffoldRunner {
	t.Helper()

	opts := &config.SkaffoldOptions{
		Trigger: "polling",
	}

	pipeline := &latest.SkaffoldPipeline{}
	defaults.Set(pipeline)

	runner, err := NewForConfig(opts, pipeline)

	testutil.CheckError(t, false, err)

	runner.Builder = testBench
	runner.Syncer = testBench
	runner.Tester = testBench
	runner.Deployer = testBench

	return runner
}

func TestNewForConfig(t *testing.T) {
	restore := testutil.SetupFakeKubernetesContext(t, api.Config{CurrentContext: "cluster1"})
	defer restore()

	var tests = []struct {
		description      string
		pipeline         *latest.SkaffoldPipeline
		shouldErr        bool
		expectedBuilder  build.Builder
		expectedTester   test.Tester
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			pipeline: &latest.SkaffoldPipeline{
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
			pipeline: &latest.SkaffoldPipeline{
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
			description: "unknown builder",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{},
			},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown tagger",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				}},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown deployer",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := NewForConfig(&config.SkaffoldOptions{
				Trigger: "polling",
			}, test.pipeline)

			testutil.CheckError(t, test.shouldErr, err)
			if cfg != nil {
				b, _t, d := WithTimings(test.expectedBuilder, test.expectedTester, test.expectedDeployer)

				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, b, cfg.Builder)
				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, _t, cfg.Tester)
				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, d, cfg.Deployer)
			}
		})
	}
}

func TestRun(t *testing.T) {
	var tests = []struct {
		description     string
		testBench       *TestBench
		shouldErr       bool
		expectedActions []Actions
	}{
		{
			description: "run no error",
			testBench:   &TestBench{},
			expectedActions: []Actions{{
				Built:    []string{"img:1"},
				Tested:   []string{"img:1"},
				Deployed: []string{"img:1"},
			}},
		},
		{
			description:     "run build error",
			testBench:       &TestBench{buildErrors: []error{errors.New("")}},
			shouldErr:       true,
			expectedActions: []Actions{{}},
		},
		{
			description: "run test error",
			testBench:   &TestBench{testErrors: []error{errors.New("")}},
			shouldErr:   true,
			expectedActions: []Actions{{
				Built: []string{"img:1"},
			}},
		},
		{
			description: "run deploy error",
			testBench:   &TestBench{deployErrors: []error{errors.New("")}},
			shouldErr:   true,
			expectedActions: []Actions{{
				Built:  []string{"img:1"},
				Tested: []string{"img:1"},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := createRunner(t, test.testBench)

			err := runner.Run(context.Background(), ioutil.Discard, []*latest.Artifact{{
				ImageName: "img",
			}})

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedActions, test.testBench.Actions())
		})
	}
}
