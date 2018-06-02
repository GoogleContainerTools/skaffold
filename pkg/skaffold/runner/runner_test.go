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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
	clientgo "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type TestBuilder struct {
	built  []build.Build
	errors []error
}

func (t *TestBuilder) Build(ctx context.Context, w io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact) ([]build.Build, error) {
	if len(t.errors) > 0 {
		err := t.errors[0]
		t.errors = t.errors[1:]
		return nil, err
	}

	var builds []build.Build

	for _, artifact := range artifacts {
		builds = append(builds, build.Build{
			ImageName: artifact.ImageName,
		})
	}

	t.built = builds
	return builds, nil
}

type TestDeployer struct {
	deployed []build.Build
	err      error
}

func (t *TestDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

func (t *TestDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Build) error {
	if t.err != nil {
		return t.err
	}

	t.deployed = builds
	return nil
}

func (t *TestDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

func resetClient()                               { kubernetes.Client = kubernetes.GetClientset }
func fakeGetClient() (clientgo.Interface, error) { return fake.NewSimpleClientset(), nil }

type TestWatcher struct {
	changes [][]string
}

func NewWatcherFactory(err error, changes ...[]string) watch.WatcherFactory {
	return func([]string) (watch.Watcher, error) {
		return &TestWatcher{
			changes: changes,
		}, err
	}
}

func (t *TestWatcher) Start(context context.Context, out io.Writer, onChange func([]string) error) error {
	for _, change := range t.changes {
		onChange(change)
	}
	return nil
}

func TestNewForConfig(t *testing.T) {
	var tests = []struct {
		description      string
		config           *v1alpha2.SkaffoldConfig
		shouldErr        bool
		expectedBuilder  build.Builder
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			config: &config.SkaffoldConfig{
				Build: v1alpha2.BuildConfig{
					TagPolicy: v1alpha2.TagPolicy{ShaTagger: &v1alpha2.ShaTagger{}},
					BuildType: v1alpha2.BuildType{
						LocalBuild: &v1alpha2.LocalBuild{},
					},
				},
				Deploy: v1alpha2.DeployConfig{
					DeployType: v1alpha2.DeployType{
						KubectlDeploy: &v1alpha2.KubectlDeploy{},
					},
				},
			},
			expectedBuilder:  &build.LocalBuilder{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "bad tagger config",
			config: &v1alpha2.SkaffoldConfig{
				Build: v1alpha2.BuildConfig{
					TagPolicy: v1alpha2.TagPolicy{},
					BuildType: v1alpha2.BuildType{
						LocalBuild: &v1alpha2.LocalBuild{},
					},
				},
				Deploy: v1alpha2.DeployConfig{
					DeployType: v1alpha2.DeployType{
						KubectlDeploy: &v1alpha2.KubectlDeploy{},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "unknown builder",
			config: &v1alpha2.SkaffoldConfig{
				Build: v1alpha2.BuildConfig{},
			},
			shouldErr:        true,
			expectedBuilder:  &build.LocalBuilder{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown tagger",
			config: &config.SkaffoldConfig{
				Build: v1alpha2.BuildConfig{
					TagPolicy: v1alpha2.TagPolicy{},
					BuildType: v1alpha2.BuildType{
						LocalBuild: &v1alpha2.LocalBuild{},
					},
				}},
			shouldErr:        true,
			expectedBuilder:  &build.LocalBuilder{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown deployer",
			config: &config.SkaffoldConfig{
				Build: v1alpha2.BuildConfig{
					TagPolicy: v1alpha2.TagPolicy{ShaTagger: &v1alpha2.ShaTagger{}},
					BuildType: v1alpha2.BuildType{
						LocalBuild: &v1alpha2.LocalBuild{},
					},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := NewForConfig(&config.SkaffoldOptions{}, test.config)

			testutil.CheckError(t, test.shouldErr, err)
			if cfg != nil {
				b, d := WithTimings(test.expectedBuilder, test.expectedDeployer)

				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, b, cfg.Builder)
				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, d, cfg.Deployer)
			}
		})
	}
}

func TestRun(t *testing.T) {
	var tests = []struct {
		description string
		config      *config.SkaffoldConfig
		builder     build.Builder
		deployer    deploy.Deployer
		shouldErr   bool
	}{
		{
			description: "run no error",
			config:      &v1alpha2.SkaffoldConfig{},
			builder:     &TestBuilder{},
			deployer:    &TestDeployer{},
		},
		{
			description: "run build error",
			config:      &v1alpha2.SkaffoldConfig{},
			builder: &TestBuilder{
				errors: []error{fmt.Errorf("")},
			},
			shouldErr: true,
		},
		{
			description: "run deploy error",
			config: &v1alpha2.SkaffoldConfig{
				Build: v1alpha2.BuildConfig{
					Artifacts: []*v1alpha2.Artifact{
						{
							ImageName: "test",
						},
					},
				},
			},
			builder: &TestBuilder{},
			deployer: &TestDeployer{
				err: fmt.Errorf(""),
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := &SkaffoldRunner{
				Builder:  test.builder,
				Deployer: test.deployer,
				opts:     &config.SkaffoldOptions{},
				Tagger:   &tag.ChecksumTagger{},
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
		watcherFactory watch.WatcherFactory
		shouldErr      bool
	}{
		{
			description: "fails to build the first time",
			builder: &TestBuilder{
				errors: []error{fmt.Errorf("")},
			},
			watcherFactory: NewWatcherFactory(nil),
			shouldErr:      true,
		},
		{
			description: "ignore subsequent build errors",
			builder: &TestBuilder{
				errors: []error{nil, fmt.Errorf("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil),
		},
		{
			description:    "bad watch dev mode",
			builder:        &TestBuilder{},
			watcherFactory: NewWatcherFactory(fmt.Errorf("")),
			shouldErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := &SkaffoldRunner{
				Builder:              test.builder,
				Deployer:             &TestDeployer{},
				opts:                 &config.SkaffoldOptions{},
				Tagger:               &tag.ChecksumTagger{},
				DependencyMapFactory: build.NewDependencyMap,
				WatcherFactory:       test.watcherFactory,
			}
			err := runner.Dev(context.Background(), ioutil.Discard, nil)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestBuildAndDeployAllArtifacts(t *testing.T) {
	kubernetes.Client = fakeGetClient
	defer resetClient()

	builder := &TestBuilder{}
	deployer := &TestDeployer{}
	artifacts := []*v1alpha2.Artifact{
		{ImageName: "image1"},
		{ImageName: "image2"},
	}
	pathToArtifacts := map[string][]*v1alpha2.Artifact{
		"path1": artifacts[0:1],
		"path2": artifacts[1:],
	}

	runner := &SkaffoldRunner{
		opts:     &config.SkaffoldOptions{},
		Builder:  builder,
		Deployer: deployer,
		DependencyMapFactory: func(artifacts []*v1alpha2.Artifact) (*build.DependencyMap, error) {
			return build.NewExplicitDependencyMap(artifacts, pathToArtifacts), nil
		},
	}

	ctx := context.Background()

	// All artifacts are changed
	runner.WatcherFactory = NewWatcherFactory(nil, []string{"path1", "path2"})
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
	runner.WatcherFactory = NewWatcherFactory(nil, []string{"path2"})
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
