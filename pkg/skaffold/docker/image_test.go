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
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
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
			shouldErr:   true,
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

var (
	digest    = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	digestOne = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	image     = fmt.Sprintf("image@%s", digest)
	imageOne  = fmt.Sprintf("image1@%s", digestOne)
)

func TestFindTaggedImageByDigest(t *testing.T) {
	tests := []struct {
		name           string
		digest         string
		imageSummaries []types.ImageSummary
		expected       string
	}{
		{
			name:   "one image id exists",
			digest: digest,
			imageSummaries: []types.ImageSummary{
				{
					RepoTags:    []string{"image:mytag"},
					RepoDigests: []string{image},
				},
				{
					RepoTags:    []string{"image1:latest"},
					RepoDigests: []string{imageOne},
				},
			},
			expected: "image:mytag",
		},
		{
			name:   "no image id exists",
			digest: "dne",
			imageSummaries: []types.ImageSummary{
				{
					RepoTags:    []string{"image:mytag"},
					RepoDigests: []string{image},
				},
				{
					RepoTags:    []string{"image:mytag"},
					RepoDigests: []string{image},
				},
			},
			expected: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &testutil.FakeAPIClient{
				ImageSummaries: test.imageSummaries,
			}

			localDocker := &localDaemon{
				apiClient: api,
			}

			actual, err := localDocker.FindTaggedImage(context.Background(), "", test.digest)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, actual)
		})
	}
}

func TestImageExists(t *testing.T) {
	client, _ := NewAPIClient()
	t.Log(client.ImageExists(context.Background(), "somethingranodm"))
	tests := []struct {
		name            string
		tagToImageID    map[string]string
		image           string
		errImageInspect bool
		expected        bool
	}{
		{
			name:         "image exists",
			image:        "image:tag",
			tagToImageID: map[string]string{"image:tag": "imageID"},
			expected:     true,
		}, {
			name:            "image does not exist",
			image:           "dne",
			errImageInspect: true,
			tagToImageID:    map[string]string{"image:tag": "imageID"},
		}, {
			name:            "error getting image",
			tagToImageID:    map[string]string{"image:tag": "imageID"},
			errImageInspect: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &testutil.FakeAPIClient{
				ErrImageInspect: test.errImageInspect,
				TagToImageID:    test.tagToImageID,
			}

			localDocker := &localDaemon{
				apiClient: api,
			}

			actual := localDocker.ImageExists(context.Background(), test.image)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}

func TestRepoDigest(t *testing.T) {
	tests := []struct {
		name            string
		image           string
		tagToImageID    map[string]string
		repoDigests     []string
		errImageInspect bool
		shouldErr       bool
		expected        string
	}{
		{
			name:         "repo digest exists",
			image:        "image:tag",
			tagToImageID: map[string]string{"image:tag": "image", "image1:tag": "image1"},
			repoDigests:  []string{"repoDigest", "repoDigest1"},
			expected:     "repoDigest",
		},
		{
			name:         "repo digest does not exist",
			image:        "image",
			tagToImageID: map[string]string{},
			repoDigests:  []string{},
			shouldErr:    true,
		},
		{
			name:            "err getting repo digest",
			image:           "image:tag",
			errImageInspect: true,
			shouldErr:       true,
			tagToImageID:    map[string]string{"image:tag": "image", "image1:tag": "image1"},
			repoDigests:     []string{"repoDigest", "repoDigest1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &testutil.FakeAPIClient{
				ErrImageInspect: test.errImageInspect,
				TagToImageID:    test.tagToImageID,
				RepoDigests:     test.repoDigests,
			}

			localDocker := &localDaemon{
				apiClient: api,
			}

			actual, err := localDocker.RepoDigest(context.Background(), test.image)
			testutil.CheckError(t, test.shouldErr, err)
			if test.shouldErr {
				return
			}
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, actual)
		})
	}
}
func TestFindTaggedImageByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		imageSummaries []types.ImageSummary
		expected       string
	}{
		{
			name: "one image id exists",
			id:   "imageid",
			imageSummaries: []types.ImageSummary{
				{
					RepoTags: []string{"image1", "image2"},
					ID:       "something",
				},
				{
					RepoTags: []string{"image3"},
					ID:       "imageid",
				},
			},
			expected: "image3",
		},
		{
			name: "multiple image ids exist",
			id:   "imageid",
			imageSummaries: []types.ImageSummary{
				{
					RepoTags: []string{"image1", "image2"},
					ID:       "something",
				},
				{
					RepoTags: []string{"image3", "image4"},
					ID:       "imageid",
				},
			},
			expected: "image3",
		},
		{
			name: "no image id exists",
			id:   "imageid",
			imageSummaries: []types.ImageSummary{
				{
					RepoTags: []string{"image1", "image2"},
					ID:       "something",
				},
				{
					RepoTags: []string{"image3"},
					ID:       "somethingelse",
				},
			},
			expected: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &testutil.FakeAPIClient{
				ImageSummaries: test.imageSummaries,
			}

			localDocker := &localDaemon{
				apiClient: api,
			}

			actual, err := localDocker.FindTaggedImage(context.Background(), test.id, "")
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, actual)
		})
	}
}
