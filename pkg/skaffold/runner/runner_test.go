/*
Copyright 2018 Google LLC

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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
	clientgo "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type TestBuilder struct {
	result []build.Build
	err    error
}

func (t *TestBuilder) Build(context.Context, io.Writer, tag.Tagger, []*v1alpha2.Artifact) ([]build.Build, error) {
	return t.result, t.err
}

type TestBuildAll struct {
}

func (t *TestBuildAll) Build(ctx context.Context, w io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact) ([]build.Build, error) {
	var builds []build.Build

	for _, artifact := range artifacts {
		builds = append(builds, build.Build{
			ImageName: artifact.ImageName,
		})
	}

	return builds, nil
}

type TestDeployer struct {
	err error
}

func (t *TestDeployer) Deploy(context.Context, io.Writer, []build.Build) error {
	return t.err
}

func (t *TestDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

func (t *TestDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

type TestDeployAll struct {
	deployed []build.Build
}

func (t *TestDeployAll) Dependencies() ([]string, error) {
	return nil, nil
}

func (t *TestDeployAll) Deploy(ctx context.Context, w io.Writer, builds []build.Build) error {
	t.deployed = builds
	return nil
}

func (t *TestDeployAll) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

type TestTagger struct {
	out string
	err error
}

func (t *TestTagger) GenerateFullyQualifiedImageName(_ string, _ *tag.TagOptions) (string, error) {
	return t.out, t.err
}

func resetClient()                                { kubernetesClient = kubernetes.GetClientset }
func fakeGetClient() (clientgo.Interface, error)  { return fake.NewSimpleClientset(), nil }
func errorGetClient() (clientgo.Interface, error) { return nil, fmt.Errorf("") }

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

func (t *TestWatcher) Start(context context.Context, onChange func([]string)) error {
	for _, change := range t.changes {
		onChange(change)
	}
	return nil
}

type TestChanges struct {
	changes [][]*v1alpha2.Artifact
}

func (t *TestChanges) OnChange(action func(artifacts []*v1alpha2.Artifact)) {
	for _, artifacts := range t.changes {
		action(artifacts)
	}
}

func TestNewForConfig(t *testing.T) {
	kubernetesClient = fakeGetClient
	defer resetClient()
	var tests = []struct {
		description string
		config      *v1alpha2.SkaffoldConfig
		shouldErr   bool
		expected    interface{}
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
			expected: &build.LocalBuilder{},
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
			shouldErr: true,
			expected:  &build.LocalBuilder{},
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
			shouldErr: true,
			expected:  &build.LocalBuilder{},
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
			cfg, err := NewForConfig(&config.SkaffoldOptions{}, test.config, ioutil.Discard)
			testutil.CheckError(t, test.shouldErr, err)
			if cfg != nil {
				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, test.expected, cfg.Builder)
			}
		})
	}
}

func TestRun(t *testing.T) {
	client, _ := fakeGetClient()
	var tests = []struct {
		description string
		runner      *SkaffoldRunner
		shouldErr   bool
	}{
		{
			description: "run no error",
			runner: &SkaffoldRunner{
				config:     &v1alpha2.SkaffoldConfig{},
				Builder:    &TestBuilder{},
				kubeclient: client,
				opts:       &config.SkaffoldOptions{},
				Tagger:     &tag.ChecksumTagger{},
				Deployer:   &TestDeployer{},
				out:        ioutil.Discard,
			},
		},
		{
			description: "run build error",
			runner: &SkaffoldRunner{
				config:     &v1alpha2.SkaffoldConfig{},
				kubeclient: client,
				Builder: &TestBuilder{
					err: fmt.Errorf(""),
				},
				opts:   &config.SkaffoldOptions{},
				Tagger: &tag.ChecksumTagger{},
				out:    ioutil.Discard,
			},
			shouldErr: true,
		},
		{
			description: "run deploy error",
			runner: &SkaffoldRunner{
				Deployer: &TestDeployer{
					err: fmt.Errorf(""),
				},
				config: &v1alpha2.SkaffoldConfig{
					Build: v1alpha2.BuildConfig{
						Artifacts: []*v1alpha2.Artifact{
							{
								ImageName: "test",
							},
						},
					},
				},
				opts:       &config.SkaffoldOptions{},
				kubeclient: client,
				Tagger:     &tag.ChecksumTagger{},
				Builder:    &TestBuilder{},
				out:        ioutil.Discard,
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := test.runner.Run(context.Background())

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestDev(t *testing.T) {
	client, _ := fakeGetClient()
	var tests = []struct {
		description string
		runner      *SkaffoldRunner
		shouldErr   bool
	}{
		{
			description: "run dev mode",
			runner: &SkaffoldRunner{
				config:     &v1alpha2.SkaffoldConfig{},
				kubeclient: client,
				Builder: &TestBuilder{
					result: []build.Build{
						{
							ImageName: "test",
							Tag:       "test:tag",
						},
					},
				},
				Deployer:       &TestDeployer{},
				WatcherFactory: NewWatcherFactory(nil, []string{}),
				opts:           &config.SkaffoldOptions{},
				Tagger:         &tag.ChecksumTagger{},
				out:            ioutil.Discard,
			},
		},
		{
			description: "run dev mode build error, continue",
			runner: &SkaffoldRunner{
				config:     &v1alpha2.SkaffoldConfig{},
				kubeclient: client,
				Builder: &TestBuilder{
					err: fmt.Errorf(""),
				},
				Deployer:       &TestDeployer{},
				Tagger:         &TestTagger{},
				WatcherFactory: NewWatcherFactory(nil, []string{}),
				opts:           &config.SkaffoldOptions{},
				out:            ioutil.Discard,
			},
		},
		{
			description: "bad watch dev mode",
			runner: &SkaffoldRunner{
				config:         &v1alpha2.SkaffoldConfig{},
				kubeclient:     client,
				Builder:        &TestBuilder{},
				Deployer:       &TestDeployer{},
				WatcherFactory: NewWatcherFactory(fmt.Errorf("")),
				opts:           &config.SkaffoldOptions{},
				Tagger:         &tag.ChecksumTagger{},
				out:            ioutil.Discard,
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := test.runner.Dev(context.Background())

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestBuildAndDeployAllArtifacts(t *testing.T) {
	kubeclient, _ := fakeGetClient()
	builder := &TestBuildAll{}
	deployer := &TestDeployAll{}

	runner := &SkaffoldRunner{
		opts:       &config.SkaffoldOptions{},
		kubeclient: kubeclient,
		Builder:    builder,
		Deployer:   deployer,
		out:        ioutil.Discard,
	}

	ctx := context.Background()

	// Build all artifacts
	bRes, err := runner.buildAndDeploy(ctx, []*v1alpha2.Artifact{
		{ImageName: "image1"},
		{ImageName: "image2"},
	}, nil)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(bRes) != 2 {
		t.Errorf("Expected 2 artifacts to be built. Got %d", len(bRes))
	}
	if len(deployer.deployed) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed))
	}

	// Rebuild only one
	bRes, err = runner.buildAndDeploy(ctx, []*v1alpha2.Artifact{
		{ImageName: "image2"},
	}, nil)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(bRes) != 1 {
		t.Errorf("Expected 1 artifact to be built. Got %d", len(bRes))
	}
	if len(deployer.deployed) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed))
	}
}
