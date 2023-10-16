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
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPush(t *testing.T) {
	tests := []struct {
		description    string
		imageName      string
		api            *testutil.FakeAPIClient
		expectedDigest string
		shouldErr      bool
	}{
		{
			description:    "push",
			imageName:      "gcr.io/scratchman",
			api:            (&testutil.FakeAPIClient{}).Add("gcr.io/scratchman", "sha256:imageIDabcab"),
			expectedDigest: "sha256:bb1f952848763dd1f8fcf14231d7a4557775abf3c95e588561bc7a478c94e7e0",
		},
		{
			description: "stream error",
			imageName:   "gcr.io/imthescratchman",
			api: &testutil.FakeAPIClient{
				ErrStream: true,
			},
			shouldErr: true,
		},
		{
			description: "image push error",
			imageName:   "gcr.io/skibabopbadopbop",
			api: &testutil.FakeAPIClient{
				ErrImagePush: true,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&DefaultAuthHelper, testAuthHelper{})

			localDocker := NewLocalDaemon(test.api, nil, false, nil)
			digest, err := localDocker.Push(context.Background(), io.Discard, test.imageName)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedDigest, digest)
		})
	}
}

func TestDoNotPushAlreadyPushed(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&DefaultAuthHelper, testAuthHelper{})

		api := &testutil.FakeAPIClient{}
		api.Add("image", "sha256:imageIDabcab")
		localDocker := NewLocalDaemon(api, nil, false, nil)

		digest, err := localDocker.Push(context.Background(), io.Discard, "image")
		t.CheckNoError(err)
		t.CheckDeepEqual("sha256:bb1f952848763dd1f8fcf14231d7a4557775abf3c95e588561bc7a478c94e7e0", digest)

		// Images already pushed don't need being pushed.
		api.ErrImagePush = true

		digest, err = localDocker.Push(context.Background(), io.Discard, "image")
		t.CheckNoError(err)
		t.CheckDeepEqual("sha256:bb1f952848763dd1f8fcf14231d7a4557775abf3c95e588561bc7a478c94e7e0", digest)
	})
}

func TestBuild(t *testing.T) {
	tests := []struct {
		description   string
		env           map[string]string
		api           *testutil.FakeAPIClient
		workspace     string
		artifact      *latest.DockerArtifact
		expected      types.ImageBuildOptions
		mode          config.RunMode
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
			mode: config.RunModes.Dev,
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
					"k2": util.Ptr("value2"),
					"k3": util.Ptr("{{.VALUE3}}"),
				},
				CacheFrom:   []string{"from-1"},
				Target:      "target",
				NetworkMode: "None",
				NoCache:     true,
				PullParent:  true,
			},
			mode: config.RunModes.Dev,
			expected: types.ImageBuildOptions{
				Tags:       []string{"finalimage"},
				Dockerfile: "Dockerfile",
				BuildArgs: map[string]*string{
					"k1": nil,
					"k2": util.Ptr("value2"),
					"k3": util.Ptr("value3"),
				},
				CacheFrom:   []string{"from-1"},
				AuthConfigs: allAuthConfig,
				Target:      "target",
				NetworkMode: "none",
				NoCache:     true,
				PullParent:  true,
			},
		},
		{
			description: "bad image build",
			api: &testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			mode:          config.RunModes.Dev,
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
			mode:          config.RunModes.Dev,
			artifact:      &latest.DockerArtifact{},
			shouldErr:     true,
			expectedError: "unable to stream build output",
		},
		{
			description: "bad build arg template",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key": util.Ptr("{{INVALID"),
				},
			},
			mode:          config.RunModes.Dev,
			shouldErr:     true,
			expectedError: `function "INVALID" not defined`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&DefaultAuthHelper, testAuthHelper{})
			t.Override(&EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
				return util.EvaluateEnvTemplateMap(args)
			})
			t.SetEnvs(test.env)

			localDocker := NewLocalDaemon(test.api, nil, false, nil)
			opts := BuildOptions{Tag: "finalimage", Mode: test.mode}
			_, err := localDocker.Build(context.Background(), io.Discard, test.workspace, "final-image", test.artifact, opts)

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
	tests := []struct {
		description string
		ref         string
		api         *testutil.FakeAPIClient
		expected    string
		shouldErr   bool
	}{
		{
			description: "find by tag",
			ref:         "identifier:latest",
			api:         (&testutil.FakeAPIClient{}).Add("identifier:latest", "sha256:123abc"),
			expected:    "sha256:123abc",
		},
		{
			description: "find by imageID",
			ref:         "sha256:123abc",
			api:         (&testutil.FakeAPIClient{}).Add("identifier:latest", "sha256:123abc"),
			expected:    "sha256:123abc",
		},
		{
			description: "image inspect error",
			ref:         "test",
			api: &testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			shouldErr: true,
		},
		{
			description: "not found",
			ref:         "somethingelse",
			api:         &testutil.FakeAPIClient{},
			expected:    "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			localDocker := NewLocalDaemon(test.api, nil, false, nil)

			imageID, err := localDocker.ImageID(context.Background(), test.ref)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, imageID)
			if test.shouldErr {
				if e, ok := err.(sErrors.Error); ok {
					t.CheckDeepEqual(e.StatusCode(), proto.StatusCode_BUILD_DOCKER_GET_DIGEST_ERR)
				} else {
					t.Error("expected to be of type actionable err not found")
				}
			}
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
					"key1": util.Ptr("value1"),
					"key2": nil,
					"key3": util.Ptr("{{.FOO}}"),
				},
			},
			env:  []string{"FOO=bar"},
			want: []string{"--build-arg", "key1=value1", "--build-arg", "key2", "--build-arg", "key3=bar"},
		},
		{
			description: "invalid build arg",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key": util.Ptr("{{INVALID"),
				},
			},
			shouldErr: true,
		},
		{
			description: "add host",
			artifact: &latest.DockerArtifact{
				AddHost: []string{"1.gcr.io:127.0.0.1", "2.gcr.io:127.0.0.1"},
			},
			want: []string{"--add-host", "1.gcr.io:127.0.0.1", "--add-host", "2.gcr.io:127.0.0.1"},
		},
		{
			description: "cache from",
			artifact: &latest.DockerArtifact{
				CacheFrom: []string{"gcr.io/foo/bar", "baz:latest"},
			},
			want: []string{"--cache-from", "gcr.io/foo/bar", "--cache-from", "baz:latest"},
		},
		{
			description: "additional CLI flags",
			artifact: &latest.DockerArtifact{
				CliFlags: []string{"--foo", "--bar"},
			},
			want: []string{"--foo", "--bar"},
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
			description: "pullParent",
			artifact: &latest.DockerArtifact{
				PullParent: true,
			},
			want: []string{"--pull"},
		},
		{
			description: "squash",
			artifact: &latest.DockerArtifact{
				Squash: true,
			},
			want: []string{"--squash"},
		},
		{
			description: "secret with no source",
			artifact: &latest.DockerArtifact{
				Secrets: []*latest.DockerSecret{
					{ID: "mysecret"},
				},
			},
			want: []string{"--secret", "id=mysecret"},
		},
		{
			description: "secret with file source",
			artifact: &latest.DockerArtifact{
				Secrets: []*latest.DockerSecret{
					{ID: "mysecret", Source: "foo.src"},
				},
			},
			want: []string{"--secret", "id=mysecret,src=foo.src"},
		},
		{
			description: "secret with file source in home directory",
			artifact: &latest.DockerArtifact{
				Secrets: []*latest.DockerSecret{
					{ID: "mysecret", Source: "~/foo.src"},
				},
			},
			want: []string{"--secret", fmt.Sprintf("id=mysecret,src=%s", util.ExpandHomePath("~/foo.src"))},
		},
		{
			description: "secret with env source",
			artifact: &latest.DockerArtifact{
				Secrets: []*latest.DockerSecret{
					{ID: "mysecret", Env: "FOO"},
				},
			},
			want: []string{"--secret", "id=mysecret,env=FOO"},
		},
		{
			description: "multiple secrets",
			artifact: &latest.DockerArtifact{
				Secrets: []*latest.DockerSecret{
					{ID: "mysecret", Source: "foo.src"},
					{ID: "anothersecret", Source: "bar.src"},
				},
			},
			want: []string{"--secret", "id=mysecret,src=foo.src", "--secret", "id=anothersecret,src=bar.src"},
		},
		{
			description: "ssh with no source",
			artifact: &latest.DockerArtifact{
				SSH: "default",
			},
			want: []string{"--ssh", "default"},
		},
		{
			description: "all",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key1": util.Ptr("value1"),
				},
				CacheFrom:   []string{"foo"},
				Target:      "stage1",
				NetworkMode: "None",
				CliFlags:    []string{"--foo", "--bar"},
				PullParent:  true,
			},
			want: []string{"--build-arg", "key1=value1", "--cache-from", "foo", "--foo", "--bar", "--target", "stage1", "--network", "none", "--pull"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.env })
			args, err := util.EvaluateEnvTemplateMap(test.artifact.BuildArgs)
			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				return
			}

			result, err := ToCLIBuildArgs(test.artifact, args)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.want, result)
			}
		})
	}
}

func TestImageExists(t *testing.T) {
	tests := []struct {
		description string
		api         *testutil.FakeAPIClient
		image       string
		expected    bool
	}{
		{
			description: "image exists",
			image:       "image:tag",
			api:         (&testutil.FakeAPIClient{}).Add("image:tag", "imageID"),
			expected:    true,
		}, {
			description: "image does not exist",
			image:       "dne",
			api: (&testutil.FakeAPIClient{
				ErrImageInspect: true,
			}).Add("image:tag", "imageID"),
		}, {
			description: "error getting image",
			api: (&testutil.FakeAPIClient{
				ErrImageInspect: true,
			}).Add("image:tag", "imageID"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			localDocker := NewLocalDaemon(test.api, nil, false, nil)

			actual := localDocker.ImageExists(context.Background(), test.image)

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestConfigFile(t *testing.T) {
	api := (&testutil.FakeAPIClient{}).Add("gcr.io/image", "sha256:imageIDabcab")

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
		CommonAPIClient: (&testutil.FakeAPIClient{}).Add("gcr.io/image", "sha256:imageIDabcab"),
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

func TestTagWithImageID(t *testing.T) {
	tests := []struct {
		description string
		imageName   string
		imageID     string
		expected    string
		shouldErr   bool
	}{
		{
			description: "success",
			imageName:   "ref",
			imageID:     "sha256:imageID",
			expected:    "ref:imageID",
		},
		{
			description: "ignore tag",
			imageName:   "ref:tag",
			imageID:     "sha256:imageID",
			expected:    "ref:imageID",
		},
		{
			description: "not found",
			imageName:   "ref",
			imageID:     "sha256:unknownImageID",
			shouldErr:   true,
		},
		{
			description: "invalid",
			imageName:   "!!invalid!!",
			shouldErr:   true,
		},
		{
			description: "empty image id",
			imageName:   "ref",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			api := (&testutil.FakeAPIClient{}).Add("sha256:imageID", "sha256:imageID")

			localDocker := NewLocalDaemon(api, nil, false, nil)
			tag, err := localDocker.TagWithImageID(context.Background(), test.imageName, test.imageID)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, tag)
		})
	}
}
