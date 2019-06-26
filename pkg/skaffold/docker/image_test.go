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

package docker

import (
	"context"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
)

func TestPush(t *testing.T) {
	var tests = []struct {
		description    string
		imageName      string
		api            testutil.FakeAPIClient
		expectedDigest string
		shouldErr      bool
	}{
		{
			description: "push",
			imageName:   "gcr.io/scratchman",
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{
					"gcr.io/scratchman": "sha256:imageIDabcab",
				},
			},
			expectedDigest: "sha256:7368613235363a696d61676549446162636162e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			description: "stream error",
			imageName:   "gcr.io/imthescratchman",
			api: testutil.FakeAPIClient{
				ErrStream: true,
			},
			shouldErr: true,
		},
		{
			description: "image push error",
			imageName:   "gcr.io/skibabopbadopbop",
			api: testutil.FakeAPIClient{
				ErrImagePush: true,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&DefaultAuthHelper, testAuthHelper{})

			localDocker := &localDaemon{
				apiClient: &test.api,
			}

			digest, err := localDocker.Push(context.Background(), ioutil.Discard, test.imageName)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedDigest, digest)
		})
	}
}

func TestBuild(t *testing.T) {
	var tests = []struct {
		description   string
		env           map[string]string
		api           *testutil.FakeAPIClient
		workspace     string
		artifact      *latest.DockerArtifact
		expected      types.ImageBuildOptions
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			api:         &testutil.FakeAPIClient{},
			workspace:   ".",
			artifact:    &latest.DockerArtifact{},
			expected: types.ImageBuildOptions{
				Tags:        []string{"finalimage"},
				AuthConfigs: allAuthConfig,
			},
		},
		{
			description: "build with options",
			api:         &testutil.FakeAPIClient{},
			env: map[string]string{
				"VALUE3": "value3",
			},
			workspace: ".",
			artifact: &latest.DockerArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs: map[string]*string{
					"k1": nil,
					"k2": util.StringPtr("value2"),
					"k3": util.StringPtr("{{.VALUE3}}"),
				},
				CacheFrom:   []string{"from-1"},
				Target:      "target",
				NetworkMode: "None",
				NoCache:     true,
			},
			expected: types.ImageBuildOptions{
				Tags:       []string{"finalimage"},
				Dockerfile: "Dockerfile",
				BuildArgs: map[string]*string{
					"k1": nil,
					"k2": util.StringPtr("value2"),
					"k3": util.StringPtr("value3"),
				},
				CacheFrom:   []string{"from-1"},
				AuthConfigs: allAuthConfig,
				Target:      "target",
				NetworkMode: "none",
				NoCache:     true,
			},
		},
		{
			description: "bad image build",
			api: &testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			workspace:     ".",
			artifact:      &latest.DockerArtifact{},
			shouldErr:     true,
			expectedError: "docker build",
		},
		{
			description: "bad return reader",
			api: &testutil.FakeAPIClient{
				ErrStream: true,
			},
			workspace:     ".",
			artifact:      &latest.DockerArtifact{},
			shouldErr:     true,
			expectedError: "unable to stream build output",
		},
		{
			description: "bad build arg template",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key": util.StringPtr("{{INVALID"),
				},
			},
			shouldErr:     true,
			expectedError: `function "INVALID" not defined`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&DefaultAuthHelper, testAuthHelper{})
			t.SetEnvs(test.env)

			localDocker := &localDaemon{
				apiClient: test.api,
			}
			_, err := localDocker.Build(context.Background(), ioutil.Discard, test.workspace, test.artifact, "finalimage")

			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckNoError(err)
				t.CheckDeepEqual(test.api.Built[0], test.expected)
			}
		})
	}
}

func TestImageID(t *testing.T) {
	var tests = []struct {
		description string
		ref         string
		api         testutil.FakeAPIClient
		expected    string
		shouldErr   bool
	}{
		{
			description: "get digest",
			ref:         "identifier:latest",
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{
					"identifier:latest": "sha256:123abc",
				},
			},
			expected: "sha256:123abc",
		},
		{
			description: "image inspect error",
			ref:         "test",
			api: testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			shouldErr: true,
		},
		{
			description: "not found",
			ref:         "somethingelse",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			localDocker := &localDaemon{
				apiClient: &test.api,
			}

			imageID, err := localDocker.ImageID(context.Background(), test.ref)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, imageID)
		})
	}
}

func TestGetBuildArgs(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.DockerArtifact
		env         []string
		want        []string
		shouldErr   bool
	}{
		{
			description: "build args",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key1": util.StringPtr("value1"),
					"key2": nil,
					"key3": util.StringPtr("{{.FOO}}"),
				},
			},
			env:  []string{"FOO=bar"},
			want: []string{"--build-arg", "key1=value1", "--build-arg", "key2", "--build-arg", "key3=bar"},
		},
		{
			description: "invalid build arg",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key": util.StringPtr("{{INVALID"),
				},
			},
			shouldErr: true,
		},
		{
			description: "cache from",
			artifact: &latest.DockerArtifact{
				CacheFrom: []string{"gcr.io/foo/bar", "baz:latest"},
			},
			want: []string{"--cache-from", "gcr.io/foo/bar", "--cache-from", "baz:latest"},
		},
		{
			description: "target",
			artifact: &latest.DockerArtifact{
				Target: "stage1",
			},
			want: []string{"--target", "stage1"},
		},
		{
			description: "network mode",
			artifact: &latest.DockerArtifact{
				NetworkMode: "Bridge",
			},
			want: []string{"--network", "bridge"},
		},
		{
			description: "no-cache",
			artifact: &latest.DockerArtifact{
				NoCache: true,
			},
			want: []string{"--no-cache"},
		},
		{
			description: "all",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key1": util.StringPtr("value1"),
				},
				CacheFrom:   []string{"foo"},
				Target:      "stage1",
				NetworkMode: "None",
			},
			want: []string{"--build-arg", "key1=value1", "--cache-from", "foo", "--target", "stage1", "--network", "none"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.env })

			result, err := GetBuildArgs(test.artifact)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.want, result)
			}
		})
	}
}

func TestImageExists(t *testing.T) {
	tests := []struct {
		description     string
		tagToImageID    map[string]string
		image           string
		errImageInspect bool
		expected        bool
	}{
		{
			description:  "image exists",
			image:        "image:tag",
			tagToImageID: map[string]string{"image:tag": "imageID"},
			expected:     true,
		}, {
			description:     "image does not exist",
			image:           "dne",
			errImageInspect: true,
			tagToImageID:    map[string]string{"image:tag": "imageID"},
		}, {
			description:     "error getting image",
			tagToImageID:    map[string]string{"image:tag": "imageID"},
			errImageInspect: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			api := &testutil.FakeAPIClient{
				ErrImageInspect: test.errImageInspect,
				TagToImageID:    test.tagToImageID,
			}

			localDocker := &localDaemon{
				apiClient: api,
			}
			actual := localDocker.ImageExists(context.Background(), test.image)

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestRepoDigest(t *testing.T) {
	tests := []struct {
		description     string
		image           string
		tagToImageID    map[string]string
		repoDigests     []string
		errImageInspect bool
		shouldErr       bool
		expected        string
	}{
		{
			description:  "repo digest exists",
			image:        "image:tag",
			tagToImageID: map[string]string{"image:tag": "image", "image1:tag": "image1"},
			repoDigests:  []string{"repoDigest", "repoDigest1"},
			expected:     "repoDigest",
		},
		{
			description:  "repo digest does not exist",
			image:        "image",
			tagToImageID: map[string]string{},
			repoDigests:  []string{},
			shouldErr:    true,
		},
		{
			description:     "err getting repo digest",
			image:           "image:tag",
			errImageInspect: true,
			shouldErr:       true,
			tagToImageID:    map[string]string{"image:tag": "image", "image1:tag": "image1"},
			repoDigests:     []string{"repoDigest", "repoDigest1"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			api := &testutil.FakeAPIClient{
				ErrImageInspect: test.errImageInspect,
				TagToImageID:    test.tagToImageID,
				RepoDigests:     test.repoDigests,
			}

			localDocker := &localDaemon{
				apiClient: api,
			}
			actual, err := localDocker.RepoDigest(context.Background(), test.image)

			if test.shouldErr {
				t.CheckError(test.shouldErr, err)
			} else {
				t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
			}
		})
	}
}

func TestInsecureRegistry(t *testing.T) {
	tests := []struct {
		description        string
		image              string
		insecureRegistries map[string]bool
		insecure           bool
		shouldErr          bool
	}{
		{
			description:        "secure image",
			image:              "gcr.io/secure/image",
			insecureRegistries: map[string]bool{},
		},
		{
			description: "insecure image",
			image:       "my.insecure.registry/image",
			insecureRegistries: map[string]bool{
				"my.insecure.registry": true,
			},
			insecure: true,
		},
		{
			description: "insecure image not provided by user",
			image:       "my.insecure.registry/image",
			insecure:    true,
			shouldErr:   true,
		},
		{
			description: "secure image provided in insecure registries list",
			image:       "gcr.io/secure/image",
			insecureRegistries: map[string]bool{
				"gcr.io": true,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			called := false // variable to make sure we've called our getInsecureRegistry function

			t.Override(&getInsecureRegistryImpl, func(string) (name.Reference, error) {
				called = true
				return name.Tag{}, nil
			})
			t.Override(&getRemoteImageImpl, func(name.Reference) (v1.Image, error) {
				return random.Image(0, 0)
			})

			_, err := remoteImage(test.image, test.insecureRegistries)

			t.CheckNoError(err)
			if !test.shouldErr {
				t.CheckDeepEqual(false, test.insecure && !called)
				t.CheckDeepEqual(false, !test.insecure && called)
			}
		})
	}
}

func TestConfigFile(t *testing.T) {
	api := &testutil.FakeAPIClient{
		TagToImageID: map[string]string{
			"gcr.io/image": "sha256:imageIDabcab",
		},
	}

	localDocker := NewLocalDaemon(api, nil, false, nil)
	cfg, err := localDocker.ConfigFile(context.Background(), "gcr.io/image")

	testutil.CheckErrorAndDeepEqual(t, false, err, "sha256:imageIDabcab", cfg.Config.Image)
}

type APICallsCounter struct {
	client.CommonAPIClient
	calls int32
}

func (c *APICallsCounter) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	atomic.AddInt32(&c.calls, 1)
	return c.CommonAPIClient.ImageInspectWithRaw(ctx, image)
}

func TestConfigFileConcurrentCalls(t *testing.T) {
	api := &APICallsCounter{
		CommonAPIClient: &testutil.FakeAPIClient{
			TagToImageID: map[string]string{
				"gcr.io/image": "sha256:imageIDabcab",
			},
		},
	}

	localDocker := NewLocalDaemon(api, nil, false, nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			localDocker.ConfigFile(context.Background(), "gcr.io/image")
			wg.Done()
		}()
	}
	wg.Wait()

	// Check that the APIClient was called only once
	testutil.CheckDeepEqual(t, int32(1), atomic.LoadInt32(&api.calls))
}
