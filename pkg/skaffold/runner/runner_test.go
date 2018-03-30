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
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/schema/v1alpha2"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/watch"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
	clientgo "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type TestBuilder struct {
	res *build.BuildResult
	err error
}

func (t *TestBuilder) Build(context.Context, io.Writer, tag.Tagger, []*v1alpha1.Artifact) (*build.BuildResult, error) {
	return t.res, t.err
}

type TestBuildAll struct {
}

func (t *TestBuildAll) Build(ctx context.Context, w io.Writer, tagger tag.Tagger, artifacts []*v1alpha1.Artifact) (*build.BuildResult, error) {
	var builds []build.Build

	for _, artifact := range artifacts {
		builds = append(builds, build.Build{
			ImageName: artifact.ImageName,
		})
	}

	return &build.BuildResult{
		Builds: builds,
	}, nil
}

type TestDeployer struct {
	res *deploy.Result
	err error
}

func (t *TestDeployer) Deploy(context.Context, io.Writer, *build.BuildResult) (*deploy.Result, error) {
	return t.res, t.err
}

type TestDeployAll struct {
	deployed *build.BuildResult
}

func (t *TestDeployAll) Deploy(ctx context.Context, w io.Writer, bRes *build.BuildResult) (*deploy.Result, error) {
	t.deployed = bRes
	return &deploy.Result{}, nil
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

func (t *TestWatcher) Start(context context.Context, onChange func([]string)) {
	for _, change := range t.changes {
		onChange(change)
	}
}

type TestChanges struct {
	changes [][]*v1alpha1.Artifact
}

func (t *TestChanges) OnChange(action func(artifacts []*v1alpha1.Artifact)) {
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
					BuildType: v1alpha1.BuildType{
						LocalBuild: &v1alpha1.LocalBuild{},
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
					BuildType: v1alpha1.BuildType{
						LocalBuild: &v1alpha1.LocalBuild{},
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
					BuildType: v1alpha1.BuildType{
						LocalBuild: &v1alpha1.LocalBuild{},
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
					BuildType: v1alpha1.BuildType{
						LocalBuild: &v1alpha1.LocalBuild{},
					},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := NewForConfig(&config.SkaffoldOptions{DevMode: false}, test.config)
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
		devmode     bool
		shouldErr   bool
	}{
		{
			description: "run no error",
			runner: &SkaffoldRunner{
				config: &v1alpha2.SkaffoldConfig{},
				Builder: &TestBuilder{
					res: &build.BuildResult{},
					err: nil,
				},
				kubeclient: client,
				opts: &config.SkaffoldOptions{
					DevMode: false,
					Output:  &bytes.Buffer{},
				},
				Tagger: &tag.ChecksumTagger{},
				Deployer: &TestDeployer{
					res: &deploy.Result{},
					err: nil,
				},
				WatcherFactory: NewWatcherFactory(nil),
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
				opts: &config.SkaffoldOptions{
					DevMode: false,
					Output:  &bytes.Buffer{},
				},
				Tagger:         &tag.ChecksumTagger{},
				WatcherFactory: NewWatcherFactory(nil),
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
						Artifacts: []*v1alpha1.Artifact{
							{
								ImageName: "test",
							},
						},
					},
				},
				opts: &config.SkaffoldOptions{
					DevMode: false,
					Output:  &bytes.Buffer{},
				},
				kubeclient: client,
				Tagger:     &tag.ChecksumTagger{},
				Builder: &TestBuilder{
					res: &build.BuildResult{},
					err: nil,
				},
				WatcherFactory: NewWatcherFactory(nil),
			},
			shouldErr: true,
		},
		{
			description: "run dev mode",
			runner: &SkaffoldRunner{
				config:     &v1alpha2.SkaffoldConfig{},
				kubeclient: client,
				Builder: &TestBuilder{
					res: &build.BuildResult{
						Builds: []build.Build{
							{
								ImageName: "test",
								Tag:       "test:tag",
							},
						},
					},
				},
				Deployer:       &TestDeployer{},
				WatcherFactory: NewWatcherFactory(nil, []string{}),
				opts: &config.SkaffoldOptions{
					DevMode: true,
					Output:  &bytes.Buffer{},
				},
				Tagger: &tag.ChecksumTagger{},
			},
		},
		{
			description: "run dev mode build error, continue",
			runner: &SkaffoldRunner{
				config:     &v1alpha2.SkaffoldConfig{},
				kubeclient: client,
				Builder: &TestBuilder{
					res: &build.BuildResult{},
					err: fmt.Errorf(""),
				},
				Deployer:       &TestDeployer{},
				Tagger:         &TestTagger{},
				WatcherFactory: NewWatcherFactory(nil, []string{}),
				opts: &config.SkaffoldOptions{
					DevMode: true,
					Output:  &bytes.Buffer{},
				},
			},
		},
		{
			description: "bad watch dev mode",
			runner: &SkaffoldRunner{
				config:     &v1alpha2.SkaffoldConfig{},
				kubeclient: client,
				Builder: &TestBuilder{
					res: &build.BuildResult{},
					err: nil,
				},
				Deployer:       &TestDeployer{},
				WatcherFactory: NewWatcherFactory(fmt.Errorf("")),
				opts: &config.SkaffoldOptions{
					DevMode: true,
					Output:  &bytes.Buffer{},
				},
				Tagger: &tag.ChecksumTagger{},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := test.runner.Run()
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestBuildAndDeployAllArtifacts(t *testing.T) {
	kubeclient, _ := fakeGetClient()
	builder := &TestBuildAll{}
	deployer := &TestDeployAll{}

	runner := &SkaffoldRunner{
		opts: &config.SkaffoldOptions{
			Output: &bytes.Buffer{},
		},
		kubeclient: kubeclient,
		Builder:    builder,
		Deployer:   deployer,
	}

	ctx := context.Background()

	// Build all artifacts
	bRes, _, err := runner.buildAndDeploy(ctx, []*v1alpha1.Artifact{
		{ImageName: "image1"},
		{ImageName: "image2"},
	}, nil)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(bRes.Builds) != 2 {
		t.Errorf("Expected 2 artifacts to be built. Got %d", len(bRes.Builds))
	}
	if len(deployer.deployed.Builds) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed.Builds))
	}

	// Rebuild only one
	bRes, _, err = runner.buildAndDeploy(ctx, []*v1alpha1.Artifact{
		{ImageName: "image2"},
	}, nil)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(bRes.Builds) != 1 {
		t.Errorf("Expected 1 artifact to be built. Got %d", len(bRes.Builds))
	}
	if len(deployer.deployed.Builds) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed.Builds))
	}
}
