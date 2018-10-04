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
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
	clientgo "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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

type TestSyncer struct {
	err     error
	copies  map[string]string
	deletes []string
}

func NewTestSyncer() *TestSyncer {
	return &TestSyncer{
		copies: map[string]string{},
	}
}

func (t *TestSyncer) Sync(s *sync.Item) error {
	if t.err != nil {
		return t.err
	}
	for src, dst := range s.Copy {
		t.copies[src] = dst
	}
	for _, dst := range s.Delete {
		t.deletes = append(t.deletes, dst)
	}
	return nil
}

func resetClient()                               { kubernetes.Client = kubernetes.GetClientset }
func fakeGetClient() (clientgo.Interface, error) { return fake.NewSimpleClientset(), nil }

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

func TestNewForConfig(t *testing.T) {
	var tests = []struct {
		description      string
		config           *latest.SkaffoldConfig
		shouldErr        bool
		expectedBuilder  build.Builder
		expectedTester   test.Tester
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			config: &latest.SkaffoldConfig{
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
			config: &latest.SkaffoldConfig{
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
			config: &latest.SkaffoldConfig{
				Build: latest.BuildConfig{},
			},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown tagger",
			config: &latest.SkaffoldConfig{
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
			config: &latest.SkaffoldConfig{
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
			}, test.config)

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
		config      *latest.SkaffoldConfig
		builder     build.Builder
		tester      test.Tester
		deployer    deploy.Deployer
		shouldErr   bool
	}{
		{
			description: "run no error",
			config:      &latest.SkaffoldConfig{},
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
		},
		{
			description: "run build error",
			config:      &latest.SkaffoldConfig{},
			builder: &TestBuilder{
				errors: []error{fmt.Errorf("")},
			},
			tester:    &TestTester{},
			shouldErr: true,
		},
		{
			description: "run deploy error",
			config: &latest.SkaffoldConfig{
				Build: latest.BuildConfig{
					Artifacts: []*latest.Artifact{
						{
							ImageName: "test",
						},
					},
				},
			},
			builder: &TestBuilder{},
			tester:  &TestTester{},
			deployer: &TestDeployer{
				errors: []error{fmt.Errorf("")},
			},
			shouldErr: true,
		},
		{
			description: "run test error",
			config: &latest.SkaffoldConfig{
				Build: latest.BuildConfig{
					Artifacts: []*latest.Artifact{
						{
							ImageName: "test",
						},
					},
				},
				Test: []latest.TestCase{
					{
						ImageName:      "test",
						StructureTests: []string{"fake_file.yaml"},
					},
				},
			},
			builder: &TestBuilder{},
			tester: &TestTester{
				errors: []error{fmt.Errorf("")},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := &SkaffoldRunner{
				Builder:  test.builder,
				Tester:   test.tester,
				Deployer: test.deployer,
				Tagger:   &tag.ChecksumTagger{},
				opts:     &config.SkaffoldOptions{},
			}
			err := runner.Run(context.Background(), ioutil.Discard, test.config.Build.Artifacts)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestDev(t *testing.T) {
	kubernetes.Client = fakeGetClient
	defer resetClient()

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
				errors: []error{fmt.Errorf("")},
			},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "fails to deploy the first time",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer: &TestDeployer{
				errors: []error{fmt.Errorf("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "fails to deploy due to failed tests",
			builder:     &TestBuilder{},
			tester: &TestTester{
				errors: []error{fmt.Errorf("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "ignore subsequent build errors",
			builder: &TestBuilder{
				errors: []error{nil, fmt.Errorf("")},
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
				errors: []error{nil, fmt.Errorf("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil, nil),
		},
		{
			description:    "fail to watch files",
			builder:        &TestBuilder{},
			tester:         &TestTester{},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(fmt.Errorf(""), nil),
			shouldErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			opts := &config.SkaffoldOptions{
				WatchPollInterval: 100,
				Trigger:           "polling",
			}

			trigger, _ := watch.NewTrigger(opts)

			runner := &SkaffoldRunner{
				Builder:      test.builder,
				Tester:       test.tester,
				Deployer:     test.deployer,
				Tagger:       &tag.ChecksumTagger{},
				Trigger:      trigger,
				watchFactory: test.watcherFactory,
				opts:         opts,
				Syncer:       NewTestSyncer(),
			}
			_, err := runner.Dev(context.Background(), ioutil.Discard, nil)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestBuildAndDeployAllArtifacts(t *testing.T) {
	kubernetes.Client = fakeGetClient
	defer resetClient()

	opts := &config.SkaffoldOptions{
		Trigger: "polling",
	}
	builder := &TestBuilder{}
	tester := &TestTester{}
	deployer := &TestDeployer{}
	trigger, _ := watch.NewTrigger(opts)
	artifacts := []*latest.Artifact{
		{ImageName: "image1"},
		{ImageName: "image2"},
	}

	runner := &SkaffoldRunner{
		Builder:  builder,
		Tester:   tester,
		Deployer: deployer,
		Trigger:  trigger,
		opts:     opts,
		Syncer:   NewTestSyncer(),
	}

	ctx := context.Background()

	// Both artifacts are changed
	runner.watchFactory = NewWatcherFactory(nil, nil, []int{0, 1})
	_, err := runner.Dev(ctx, ioutil.Discard, artifacts)

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
	_, err = runner.Dev(ctx, ioutil.Discard, artifacts)

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
			})

			testutil.CheckDeepEqual(t, test.expectedMatch, match)
		})
	}
}
