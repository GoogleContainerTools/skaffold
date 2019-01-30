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

package docker

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestMain(m *testing.M) {
	// So we don't shell out to credentials helpers or try to read dockercfg
	defer func(h AuthConfigHelper) { DefaultAuthHelper = h }(DefaultAuthHelper)
	DefaultAuthHelper = testAuthHelper{}

	os.Exit(m.Run())
}

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
					"gcr.io/scratchman": "sha256:abcab",
				},
			},
			expectedDigest: "sha256:abcab",
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
		t.Run(test.description, func(t *testing.T) {
			localDocker := &localDaemon{
				apiClient: &test.api,
			}

			digest, err := localDocker.Push(context.Background(), ioutil.Discard, test.imageName)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedDigest, digest)
		})
	}
}

func TestRunBuild(t *testing.T) {
	var tests = []struct {
		description string
		expected    string
		api         testutil.FakeAPIClient
		shouldErr   bool
	}{
		{
			description: "build",
			expected:    "test",
		},
		{
			description: "bad image build",
			api: testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			shouldErr: true,
		},
		{
			description: "bad return reader",
			api: testutil.FakeAPIClient{
				ErrStream: true,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			localDocker := &localDaemon{
				apiClient: &test.api,
			}

			_, err := localDocker.Build(context.Background(), ioutil.Discard, ".", &latest.DockerArtifact{}, "finalimage")

			testutil.CheckError(t, test.shouldErr, err)
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
			expected:    "",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			localDocker := &localDaemon{
				apiClient: &test.api,
			}

			imageID, err := localDocker.ImageID(context.Background(), test.ref)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, imageID)
		})
	}
}

func TestGetBuildArgs(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.DockerArtifact
		want        []string
	}{
		{
			description: "build args",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key1": util.StringPtr("value1"),
					"key2": nil,
				},
			},
			want: []string{"--build-arg", "key1=value1", "--build-arg", "key2"},
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
			description: "all",
			artifact: &latest.DockerArtifact{
				BuildArgs: map[string]*string{
					"key1": util.StringPtr("value1"),
				},
				CacheFrom: []string{"foo"},
				Target:    "stage1",
			},
			want: []string{"--build-arg", "key1=value1", "--cache-from", "foo", "--target", "stage1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := GetBuildArgs(tt.artifact)
			if diff := cmp.Diff(result, tt.want); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", tt.want, diff)
			}
		})
	}
}
