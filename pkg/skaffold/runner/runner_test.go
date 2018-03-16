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

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/watch"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
	clientgo "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type TestBuilder struct {
	res *build.BuildResult
	err error
}

func (t *TestBuilder) Run(io.Writer, tag.Tagger, []*config.Artifact) (*build.BuildResult, error) {
	return t.res, t.err
}

type TestDeployer struct {
	res *deploy.Result
	err error
}

func (t *TestDeployer) Run(io.Writer, *build.BuildResult) (*deploy.Result, error) {
	return t.res, t.err
}

type TestTagger struct {
	out string
	err error
}

func (t *TestTagger) GenerateFullyQualifiedImageName(_ *tag.TagOptions) (string, error) {
	return t.out, t.err
}

func resetClient()                                { kubernetesClient = kubernetes.GetClientset }
func fakeGetClient() (clientgo.Interface, error)  { return fake.NewSimpleClientset(), nil }
func errorGetClient() (clientgo.Interface, error) { return nil, fmt.Errorf("") }

type TestWatcher struct {
	changes [][]*config.Artifact
}

func NewWatcherFactory(err error, changes ...[]*config.Artifact) watch.WatcherFactory {
	return func([]*config.Artifact) (watch.Watcher, error) {
		return &TestWatcher{
			changes: changes,
		}, err
	}
}

func (t *TestWatcher) Start(context context.Context, onChange func([]*config.Artifact)) {
	for _, change := range t.changes {
		onChange(change)
	}
}

type TestChanges struct {
	changes [][]*config.Artifact
}

func (t *TestChanges) OnChange(action func(artifacts []*config.Artifact)) {
	for _, artifacts := range t.changes {
		action(artifacts)
	}
}

func TestNewForConfig(t *testing.T) {
	kubernetesClient = fakeGetClient
	defer resetClient()
	var tests = []struct {
		description string
		config      *config.SkaffoldConfig
		shouldErr   bool
		expected    interface{}
	}{
		{
			description: "local builder config",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{
					TagPolicy: constants.TagStrategySha256,
					BuildType: config.BuildType{
						LocalBuild: &config.LocalBuild{},
					},
				},
				Deploy: config.DeployConfig{
					DeployType: config.DeployType{
						KubectlDeploy: &config.KubectlDeploy{},
					},
				},
			},
			expected: &build.LocalBuilder{},
		},
		{
			description: "bad tagger config",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{
					BuildType: config.BuildType{
						LocalBuild: &config.LocalBuild{},
					},
				},
				Deploy: config.DeployConfig{
					DeployType: config.DeployType{
						KubectlDeploy: &config.KubectlDeploy{},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "unknown builder",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{},
			},
			shouldErr: true,
			expected:  &build.LocalBuilder{},
		},
		{
			description: "unknown tagger",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{
					TagPolicy: "bad tag strategy",
					BuildType: config.BuildType{
						LocalBuild: &config.LocalBuild{},
					},
				}},
			shouldErr: true,
			expected:  &build.LocalBuilder{},
		},
		{
			description: "unknown deployer",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{
					TagPolicy: constants.TagStrategySha256,
					BuildType: config.BuildType{
						LocalBuild: &config.LocalBuild{},
					},
				},
				Deploy: config.DeployConfig{},
			},
			shouldErr: true,
		},
		{
			description: "nil deployer",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{
					TagPolicy: constants.TagStrategySha256,
					BuildType: config.BuildType{
						LocalBuild: &config.LocalBuild{},
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
				config: &config.SkaffoldConfig{},
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
				config:     &config.SkaffoldConfig{},
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
				config: &config.SkaffoldConfig{
					Build: config.BuildConfig{
						Artifacts: []*config.Artifact{
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
				config:     &config.SkaffoldConfig{},
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
				WatcherFactory: NewWatcherFactory(nil, []*config.Artifact{}),
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
				config:     &config.SkaffoldConfig{},
				kubeclient: client,
				Builder: &TestBuilder{
					res: &build.BuildResult{},
					err: fmt.Errorf(""),
				},
				Deployer:       &TestDeployer{},
				Tagger:         &TestTagger{},
				WatcherFactory: NewWatcherFactory(nil, []*config.Artifact{}),
				opts: &config.SkaffoldOptions{
					DevMode: true,
					Output:  &bytes.Buffer{},
				},
			},
		},
		{
			description: "bad watch dev mode",
			runner: &SkaffoldRunner{
				config:     &config.SkaffoldConfig{},
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
