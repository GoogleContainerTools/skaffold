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
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/custom"
	dockerbuilder "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

type testAuthHelper struct{}

func (t testAuthHelper) GetAuthConfig(context.Context, string) (registry.AuthConfig, error) {
	return registry.AuthConfig{}, nil
}
func (t testAuthHelper) GetAllAuthConfigs(context.Context) (map[string]registry.AuthConfig, error) {
	return nil, nil
}

type previousArtifact struct {
	ImageName string
	Tag       string
	ImageID   string
}

func TestLocalRun(t *testing.T) {
	tests := []struct {
		description       string
		api               *testutil.FakeAPIClient
		tag               string
		artifact          *latest.Artifact
		previousArtifacts []previousArtifact
		mode              config.RunMode
		expected          string
		expectedWarnings  []string
		expectedPushed    map[string]string
		expectedPruned    []string
		pushImages        bool
		shouldErr         bool
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
			expectedWarnings: []string{"cacheFrom image \"pull1\" couldn't be pulled for platform \"\"\n"},
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
		{
			description: "dev mode prunes previous artifacts",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			previousArtifacts: []previousArtifact{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:1",
					ImageID:   "sha256:0",
				},
			},
			tag:            "gcr.io/test/image:tag",
			api:            &testutil.FakeAPIClient{},
			pushImages:     false,
			mode:           config.RunModes.Dev,
			expected:       "gcr.io/test/image:1",
			expectedPruned: []string{"sha256:0"},
		},
		{
			description: "dev mode doesn't prune previous artifact if image ID is the same",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
			previousArtifacts: []previousArtifact{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:1",
					ImageID:   "sha256:1",
				},
			},
			tag:            "gcr.io/test/image:tag",
			api:            &testutil.FakeAPIClient{},
			pushImages:     false,
			mode:           config.RunModes.Dev,
			expected:       "gcr.io/test/image:1",
			expectedPruned: nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&docker.DefaultAuthHelper, testAuthHelper{})
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			imageIds := map[string]string{}
			for _, a := range test.previousArtifacts {
				imageIds[a.Tag] = a.ImageID
			}
			fDockerDaemon := &fakeDockerDaemon{
				LocalDaemon: docker.NewLocalDaemon(test.api, nil, false, nil),
				ImageIds:    imageIds,
			}
			t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return fDockerDaemon, nil
			})
			t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
				return args, nil
			})
			testEvent.InitializeState([]latest.Pipeline{{
				Deploy: latest.DeployConfig{},
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				}}})

			artifactStore := mockArtifactStore{}
			for _, a := range test.previousArtifacts {
				artifactStore[a.ImageName] = a.Tag
			}
			builder, err := NewBuilder(context.Background(), &mockBuilderContext{artifactStore: artifactStore, mode: test.mode}, &latest.LocalBuild{
				Push:        util.Ptr(test.pushImages),
				Concurrency: &constants.DefaultLocalConcurrency,
			})
			t.CheckNoError(err)
			ab := builder.Build(context.Background(), io.Discard, test.artifact)
			res, err := ab(context.Background(), io.Discard, test.artifact, test.tag, platform.Matcher{})
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, res)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
			t.CheckDeepEqual(test.expectedPushed, test.api.Pushed())
			if len(test.expectedPruned) > 0 {
				// wait for completion of the prune operation which happens in a goroutine
				numAttempts := 0
				for len(fDockerDaemon.GetPrunedImages()) == 0 && numAttempts < 10 {
					time.Sleep(10 * time.Millisecond)
					numAttempts++
					println(numAttempts)
				}
				t.CheckDeepEqual(test.expectedPruned, fDockerDaemon.PrunedImages)
			}
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
		pushFlag      config.BoolOrUndefined
		localBuild    latest.LocalBuild
		localDockerFn func(context.Context, docker.Config) (docker.LocalDaemon, error)
	}{
		{
			description: "failed to get docker client",
			localDockerFn: func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return nil, errors.New("dummy docker error")
			},
			shouldErr: true,
		},
		{
			description: "pushImages becomes cluster.PushImages when local:push and --push is not defined",
			localDockerFn: func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return dummyDaemon, nil
			},
			cluster:      config.Cluster{PushImages: true},
			expectedPush: true,
		},
		{
			description: "pushImages becomes config (local:push) when --push is not defined",
			localDockerFn: func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return dummyDaemon, nil
			},
			cluster: config.Cluster{PushImages: true},
			localBuild: latest.LocalBuild{
				Push: util.Ptr(false),
			},
			shouldErr:    false,
			expectedPush: false,
		},
		{
			description: "pushImages defined in flags (--push=false), ignores cluster.PushImages",
			localDockerFn: func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return dummyDaemon, nil
			},
			cluster:      config.Cluster{PushImages: true},
			pushFlag:     config.NewBoolOrUndefined(util.Ptr(false)),
			shouldErr:    false,
			expectedPush: false,
		},
		{
			description: "pushImages defined in flags (--push=false), ignores config (local:push)",
			localDockerFn: func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return dummyDaemon, nil
			},
			pushFlag: config.NewBoolOrUndefined(util.Ptr(false)),
			localBuild: latest.LocalBuild{
				Push: util.Ptr(true),
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

			builder, err := NewBuilder(context.Background(), &mockBuilderContext{
				local:    test.localBuild,
				cluster:  test.cluster,
				pushFlag: test.pushFlag,
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
			t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return fakeLocalDaemon(&testutil.FakeAPIClient{}), nil
			})
			t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
				return args, nil
			})

			b, err := NewBuilder(context.Background(), &mockBuilderContext{artifactStore: build.NewArtifactStore()}, &latest.LocalBuild{Concurrency: &constants.DefaultLocalConcurrency})
			t.CheckNoError(err)

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

type mockBuilderContext struct {
	runcontext.RunContext // Embedded to provide the default values.
	local                 latest.LocalBuild
	mode                  config.RunMode
	cluster               config.Cluster
	pushFlag              config.BoolOrUndefined
	artifactStore         build.ArtifactStore
	sourceDepsResolver    func() graph.SourceDependenciesCache
}

func (c *mockBuilderContext) Mode() config.RunMode {
	return c.mode
}

func (c *mockBuilderContext) GetCluster() config.Cluster {
	return c.cluster
}

func (c *mockBuilderContext) PushImages() config.BoolOrUndefined {
	return c.pushFlag
}

func (c *mockBuilderContext) ArtifactStore() build.ArtifactStore {
	return c.artifactStore
}

func (c *mockBuilderContext) SourceDependenciesResolver() graph.SourceDependenciesCache {
	if c.sourceDepsResolver != nil {
		return c.sourceDepsResolver()
	}
	return nil
}

type mockArtifactStore map[string]string

func (m mockArtifactStore) GetImageTag(imageName string) (string, bool) {
	v, ok := m[imageName]
	if !ok {
		return "", false
	}
	return v, ok
}
func (m mockArtifactStore) Record(a *latest.Artifact, tag string) { m[a.ImageName] = tag }
func (m mockArtifactStore) GetArtifacts(s []*latest.Artifact) ([]graph.Artifact, error) {
	var builds []graph.Artifact
	for _, a := range s {
		t, found := m.GetImageTag(a.ImageName)
		if !found {
			return nil, fmt.Errorf("failed to retrieve build result for image %s", a.ImageName)
		}
		builds = append(builds, graph.Artifact{ImageName: a.ImageName, Tag: t, RuntimeType: a.RuntimeType})
	}
	return builds, nil
}

type fakeDockerDaemon struct {
	docker.LocalDaemon

	ImageIds     map[string]string
	PrunedImages []string
	mu           sync.Mutex
}

func (fd *fakeDockerDaemon) ImageInspectWithRaw(_ context.Context, img string) (image.InspectResponse, []byte, error) {
	imageID := fd.ImageIds[img]
	return image.InspectResponse{
		Config: &dockerspec.DockerOCIImageConfig{},
		ID:     imageID,
	}, []byte{}, nil
}

func (fd *fakeDockerDaemon) Prune(_ context.Context, images []string, _ bool) ([]string, error) {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	fd.PrunedImages = append(fd.PrunedImages, images...)
	return fd.PrunedImages, nil
}

func (fd *fakeDockerDaemon) GetPrunedImages() []string {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	return fd.PrunedImages
}
