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

package local

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/custom"
	dockerbuilder "github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type testAuthHelper struct{}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t testAuthHelper) GetAllAuthConfigs(context.Context) (map[string]types.AuthConfig, error) {
	return nil, nil
}

func TestLocalRun(t *testing.T) {
	tests := []struct {
		description      string
		api              *testutil.FakeAPIClient
		tag              string
		artifact         *latest.Artifact
		expected         string
		expectedWarnings []string
		expectedPushed   map[string]string
		pushImages       bool
		shouldErr        bool
	}{
		{
			description: "single build (local)",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			tag:        "gcr.io/test/image:tag",
			api:        &testutil.FakeAPIClient{},
			pushImages: false,
			expected:   "gcr.io/test/image:1",
		},
		{
			description: "error getting image digest",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			tag: "gcr.io/test/image:tag",
			api: &testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			shouldErr: true,
		},
		{
			description: "single build (remote)",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			tag:        "gcr.io/test/image:tag",
			api:        &testutil.FakeAPIClient{},
			pushImages: true,
			expected:   "gcr.io/test/image:tag@sha256:51ae7fa00c92525c319404a3a6d400e52ff9372c5a39cb415e0486fe425f3165",
			expectedPushed: map[string]string{
				"gcr.io/test/image:tag": "sha256:51ae7fa00c92525c319404a3a6d400e52ff9372c5a39cb415e0486fe425f3165",
			},
		},
		{
			description: "error build",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			tag: "gcr.io/test/image:tag",
			api: &testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			shouldErr: true,
		},
		{
			description: "Don't push on build error",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			tag:        "gcr.io/test/image:tag",
			pushImages: true,
			api: &testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			shouldErr: true,
		},
		{
			description: "unknown artifact type",
			artifact:    &latest.Artifact{},
			api:         &testutil.FakeAPIClient{},
			shouldErr:   true,
		},
		{
			description: "cache-from images already pulled",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull1", "pull2"},
					},
				},
			},
			api:      (&testutil.FakeAPIClient{}).Add("pull1", "imageID1").Add("pull2", "imageID2"),
			tag:      "gcr.io/test/image:tag",
			expected: "gcr.io/test/image:1",
		},
		{
			description: "pull cache-from images",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull1", "pull2"},
					},
				},
			},
			api:      (&testutil.FakeAPIClient{}).Add("pull1", "imageid").Add("pull2", "anotherimageid"),
			tag:      "gcr.io/test/image:tag",
			expected: "gcr.io/test/image:1",
		},
		{
			description: "ignore cache-from pull error",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull1"},
					},
				},
			},
			api: (&testutil.FakeAPIClient{
				ErrImagePull: true,
			}).Add("pull1", ""),
			tag:              "gcr.io/test/image:tag",
			expected:         "gcr.io/test/image:1",
			expectedWarnings: []string{"cacheFrom image couldn't be pulled: pull1\n"},
		},
		{
			description: "error checking cache-from image",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull"},
					},
				},
			},
			api: &testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			tag:       "gcr.io/test/image:tag",
			shouldErr: true,
		},
		{
			description: "fail fast docker not found",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			tag: "gcr.io/test/image:tag",
			api: &testutil.FakeAPIClient{
				ErrVersion: true,
			},
			pushImages: false,
			shouldErr:  true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&docker.DefaultAuthHelper, testAuthHelper{})
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
				return fakeLocalDaemon(test.api), nil
			})
			t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string) (map[string]*string, error) {
				return args, nil
			})
			event.InitializeState([]latest.Pipeline{{
				Deploy: latest.DeployConfig{},
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				}}}, "", true, true, true)

			builder, err := NewBuilder(&mockConfig{},
				&latest.LocalBuild{
					Push:        util.BoolPtr(test.pushImages),
					Concurrency: &constants.DefaultLocalConcurrency,
				})
			t.CheckNoError(err)
			builder.ArtifactStore(build.NewArtifactStore())
			ab := builder.Build(context.Background(), ioutil.Discard, test.artifact)
			res, err := ab(context.Background(), ioutil.Discard, test.artifact, test.tag)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, res)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
			t.CheckDeepEqual(test.expectedPushed, test.api.Pushed())
		})
	}
}

type dummyLocalDaemon struct {
	docker.LocalDaemon
}

func TestNewBuilder(t *testing.T) {
	dummyDaemon := dummyLocalDaemon{}

	tests := []struct {
		description   string
		shouldErr     bool
		expectedPush  bool
		cluster       config.Cluster
		localBuild    latest.LocalBuild
		localDockerFn func(docker.Config) (docker.LocalDaemon, error)
	}{
		{
			description: "failed to get docker client",
			localDockerFn: func(docker.Config) (docker.LocalDaemon, error) {
				return nil, errors.New("dummy docker error")
			},
			shouldErr: true,
		},
		{
			description: "pushImages becomes cluster.PushImages when local:push is not defined",
			localDockerFn: func(docker.Config) (docker.LocalDaemon, error) {
				return dummyDaemon, nil
			},
			cluster:      config.Cluster{PushImages: true},
			expectedPush: true,
		},
		{
			description: "pushImages defined in config (local:push)",
			localDockerFn: func(docker.Config) (docker.LocalDaemon, error) {
				return dummyDaemon, nil
			},
			cluster: config.Cluster{PushImages: true},
			localBuild: latest.LocalBuild{
				Push: util.BoolPtr(false),
			},
			shouldErr:    false,
			expectedPush: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.localDockerFn != nil {
				t.Override(&docker.NewAPIClient, test.localDockerFn)
			}

			builder, err := NewBuilder(&mockConfig{
				local:   test.localBuild,
				cluster: test.cluster,
			}, &test.localBuild)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expectedPush, builder.pushImages)
			}
		})
	}
}

func TestGetArtifactBuilder(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.Artifact
		expected    string
		shouldErr   bool
	}{
		{
			description: "docker builder",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			expected: "docker",
		},
		{
			description: "jib builder",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					JibArtifact: &latest.JibArtifact{},
				},
			},
			expected: "jib",
		},
		{
			description: "buildpacks builder",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{},
				},
			},
			expected: "buildpacks",
		},
		{
			description: "bazel builder",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					BazelArtifact: &latest.BazelArtifact{},
				},
			},
			expected: "bazel",
		},
		{
			description: "custom builder",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					CustomArtifact: &latest.CustomArtifact{},
				},
			},
			expected: "custom",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
				return fakeLocalDaemon(&testutil.FakeAPIClient{}), nil
			})
			t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string) (map[string]*string, error) {
				return args, nil
			})

			b, err := NewBuilder(&mockConfig{}, &latest.LocalBuild{Concurrency: &constants.DefaultLocalConcurrency})
			t.CheckNoError(err)
			b.ArtifactStore(build.NewArtifactStore())

			builder, err := newPerArtifactBuilder(b, test.artifact)
			t.CheckNoError(err)

			switch builder.(type) {
			case *dockerbuilder.Builder:
				t.CheckDeepEqual(test.expected, "docker")
			case *bazel.Builder:
				t.CheckDeepEqual(test.expected, "bazel")
			case *buildpacks.Builder:
				t.CheckDeepEqual(test.expected, "buildpacks")
			case *custom.Builder:
				t.CheckDeepEqual(test.expected, "custom")
			case *jib.Builder:
				t.CheckDeepEqual(test.expected, "jib")
			}
		})
	}
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, false, nil)
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	local                 latest.LocalBuild
	mode                  config.RunMode
	cluster               config.Cluster
}

func (c *mockConfig) Mode() config.RunMode {
	return c.mode
}

func (c *mockConfig) GetCluster() config.Cluster {
	return c.cluster
}
