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
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

type TestBuilder struct {
	built  []build.Artifact
	errors []error
}

func (t *TestBuilder) Labels() map[string]string {
	return map[string]string{}
}

func (t *TestBuilder) Build(ctx context.Context, w io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if len(t.errors) > 0 {
		err := t.errors[0]
		t.errors = t.errors[1:]
		return nil, err
	}

	var builds []build.Artifact

	for _, artifact := range artifacts {
		builds = append(builds, build.Artifact{
			ImageName: artifact.ImageName,
		})
	}

	t.built = builds
	return builds, nil
}

type TestTester struct {
	errors []error
}

func (t *TestTester) Test(ctx context.Context, out io.Writer, builds []build.Artifact) error {
	if len(t.errors) > 0 {
		err := t.errors[0]
		t.errors = t.errors[1:]
		return err
	}
	return nil
}

func (t *TestTester) TestDependencies() ([]string, error) {
	return nil, nil
}

type TestDeployer struct {
	deployed []build.Artifact
	errors   []error
}

func (t *TestDeployer) Labels() map[string]string {
	return map[string]string{}
}

func (t *TestDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

func (t *TestDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]deploy.Artifact, error) {
	if len(t.errors) > 0 {
		err := t.errors[0]
		t.errors = t.errors[1:]
		return nil, err
	}

	t.deployed = builds
	return nil, nil
}

func (t *TestDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

type TestWatcher struct {
	changedArtifacts [][]int
	changeCallbacks  []func(watch.Events)
	events           []watch.Events
	err              error
}

func NewWatcherFactory(err error, events []watch.Events, changedArtifacts ...[]int) watch.Factory {
	return func() watch.Watcher {
		return &TestWatcher{
			changedArtifacts: changedArtifacts,
			events:           events,
			err:              err,
		}
	}
}

func (t *TestWatcher) Register(deps func() ([]string, error), onChange func(watch.Events)) error {
	t.changeCallbacks = append(t.changeCallbacks, onChange)
	return nil
}

func (t *TestWatcher) Run(ctx context.Context, trigger watch.Trigger, onChange func() error) error {
	evts := watch.Events{}
	if t.events != nil {
		evts = t.events[0]
		t.events = t.events[1:]
	}

	for _, artifactIndices := range t.changedArtifacts {
		for _, artifactIndex := range artifactIndices {
			t.changeCallbacks[artifactIndex](evts)
		}
		onChange()
	}
	return t.err
}

func createDefaultRunner(t *testing.T) *SkaffoldRunner {
	t.Helper()

	opts := &config.SkaffoldOptions{
		Trigger: "polling",
	}

	pipeline := &latest.SkaffoldPipeline{}
	pipeline.Parse(nil, true)

	runner, err := NewForConfig(opts, pipeline)

	testutil.CheckError(t, false, err)

	return runner
}

func TestNewForConfig(t *testing.T) {
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
		description string
		builder     build.Builder
		tester      test.Tester
		deployer    deploy.Deployer
		shouldErr   bool
	}{
		{
			description: "run no error",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
		},
		{
			description: "run build error",
			builder: &TestBuilder{
				errors: []error{errors.New("")},
			},
			tester:    &TestTester{},
			shouldErr: true,
		},
		{
			description: "run deploy error",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer: &TestDeployer{
				errors: []error{errors.New("")},
			},
			shouldErr: true,
		},
		{
			description: "run test error",
			builder:     &TestBuilder{},
			tester: &TestTester{
				errors: []error{errors.New("")},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := createDefaultRunner(t)
			runner.Builder = test.builder
			runner.Tester = test.tester
			runner.Deployer = test.deployer

			err := runner.Run(context.Background(), ioutil.Discard, []*latest.Artifact{{
				ImageName: "test",
			}})

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestDev(t *testing.T) {
	var tests = []struct {
		description    string
		builder        build.Builder
		tester         test.Tester
		deployer       deploy.Deployer
		watcherFactory watch.Factory
		shouldErr      bool
	}{
		{
			description: "fails to build the first time",
			builder: &TestBuilder{
				errors: []error{errors.New("")},
			},
			tester:         &TestTester{},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "fails to deploy the first time",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer: &TestDeployer{
				errors: []error{errors.New("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "fails to deploy due to failed tests",
			builder:     &TestBuilder{},
			tester: &TestTester{
				errors: []error{errors.New("")},
			},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "ignore subsequent build errors",
			builder: &TestBuilder{
				errors: []error{nil, errors.New("")},
			},
			tester:         &TestTester{},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(nil, nil, nil),
		},
		{
			description: "ignore subsequent deploy errors",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer: &TestDeployer{
				errors: []error{nil, errors.New("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil, nil),
		},
		{
			description:    "fail to watch files",
			builder:        &TestBuilder{},
			tester:         &TestTester{},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(errors.New(""), nil),
			shouldErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := createDefaultRunner(t)
			runner.Builder = test.builder
			runner.Tester = test.tester
			runner.Deployer = test.deployer
			runner.watchFactory = test.watcherFactory

			err := runner.Dev(context.Background(), ioutil.Discard, nil)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestBuildAndDeployAllArtifacts(t *testing.T) {
	builder := &TestBuilder{}
	deployer := &TestDeployer{}
	artifacts := []*latest.Artifact{
		{ImageName: "image1", Watch: util.BoolPtr(true)},
		{ImageName: "image2", Watch: util.BoolPtr(true)},
	}

	runner := createDefaultRunner(t)
	runner.Builder = builder
	runner.Deployer = deployer

	ctx := context.Background()

	// Both artifacts are changed
	runner.watchFactory = NewWatcherFactory(nil, nil, []int{0, 1})
	err := runner.Dev(ctx, ioutil.Discard, artifacts)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(builder.built) != 2 {
		t.Errorf("Expected 2 artifacts to be built. Got %d", len(builder.built))
	}
	if len(deployer.deployed) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed))
	}

	// Only one is changed
	runner.watchFactory = NewWatcherFactory(nil, nil, []int{1})
	err = runner.Dev(ctx, ioutil.Discard, artifacts)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(builder.built) != 1 {
		t.Errorf("Expected 1 artifact to be built. Got %d", len(builder.built))
	}
	if len(deployer.deployed) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed))
	}
}

func TestShouldWatch(t *testing.T) {
	var tests = []struct {
		description   string
		watch         []string
		expectedMatch bool
	}{
		{
			description:   "match all",
			watch:         nil,
			expectedMatch: true,
		},
		{
			description:   "match full name",
			watch:         []string{"domain/image"},
			expectedMatch: true,
		},
		{
			description:   "match partial name",
			watch:         []string{"image"},
			expectedMatch: true,
		},
		{
			description:   "match any",
			watch:         []string{"other", "image"},
			expectedMatch: true,
		},
		{
			description:   "no match",
			watch:         []string{"other"},
			expectedMatch: false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := &SkaffoldRunner{
				opts: &config.SkaffoldOptions{
					Watch: test.watch,
				},
			}

			match := runner.shouldWatch(&latest.Artifact{
				ImageName: "domain/image",
				Watch:     util.BoolPtr(true),
			})

			testutil.CheckDeepEqual(t, test.expectedMatch, match)
		})
	}
}
