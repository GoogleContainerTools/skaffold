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

package cache

import (
	"context"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
)

func TestRetrieveCachedArtifacts_Local(t *testing.T) {
	tests := []struct {
		description              string
		cache                    *Cache
		api                      testutil.FakeAPIClient
		hashes                   map[string]string
		artifacts                []*latest.Artifact
		expectedArtifactsToBuild []*latest.Artifact
		expectedBuildResults     []build.Artifact
		expectedTaggedImages     []string
	}{
		{
			description: "no cache",
			cache:       &Cache{},
		}, {
			description: "artifact exists locally",
			cache: &Cache{
				artifactCache: ArtifactCache{"hash": ImageDetails{
					Digest: "sha256@digest",
				}},
			},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image:hash": "image1:tag"},
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{"sha256@digest"},
						RepoTags:    []string{"image:hash"},
					},
				},
			},
			hashes:    map[string]string{"image": "hash"},
			artifacts: []*latest.Artifact{{ImageName: "image"}},
			expectedBuildResults: []build.Artifact{
				{
					ImageName: "image",
					Tag:       "image:hash",
				},
			},
		}, {
			description: "artifact exists under a different tag",
			cache: &Cache{
				artifactCache: ArtifactCache{"hash": ImageDetails{
					ID: "imageID",
				}},
				imageList: []types.ImageSummary{{
					ID:       "imageID",
					RepoTags: []string{"image:anothertag"},
				}}},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image:anothertag": "imageID"},
			},
			hashes:    map[string]string{"image": "hash"},
			artifacts: []*latest.Artifact{{ImageName: "image"}},
			expectedBuildResults: []build.Artifact{
				{
					ImageName: "image",
					Tag:       "image:hash",
				},
			},
			expectedTaggedImages: []string{"image:hash"},
		}, {
			description: "artifact doesn't exist",
			cache: &Cache{
				artifactCache: ArtifactCache{"hash": ImageDetails{
					ID: "imageID",
				}}},
			api:                      testutil.FakeAPIClient{},
			hashes:                   map[string]string{"image": "hash"},
			artifacts:                []*latest.Artifact{{ImageName: "image"}},
			expectedArtifactsToBuild: []*latest.Artifact{{ImageName: "image", WorkspaceHash: "hash"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			test.cache.localCluster = true
			test.cache.useCache = true
			t.Override(&hashForArtifact, mockHashForArtifact(test.hashes))

			test.cache.client = docker.NewLocalDaemon(&test.api, nil, false, map[string]bool{})
			actualArtifactsToBuild, actualBuildResults, err := test.cache.RetrieveCachedArtifacts(context.Background(), os.Stdout, test.artifacts)
			t.CheckError(false, err)
			t.CheckDeepEqual(actualArtifactsToBuild, test.expectedArtifactsToBuild)
			t.CheckDeepEqual(actualBuildResults, test.expectedBuildResults)
			t.CheckDeepEqual(test.expectedTaggedImages, test.api.Tagged)
		})
	}
}

func TestRetrieveCachedArtifacts_Remote(t *testing.T) {
	tests := []struct {
		description               string
		cache                     *Cache
		api                       testutil.FakeAPIClient
		hashes                    map[string]string
		artifacts                 []*latest.Artifact
		targetImageExistsRemotely bool
		expectedArtifactsToBuild  []*latest.Artifact
		expectedBuildResults      []build.Artifact
		expectedTaggedImages      []string
		expectedPushedImages      []string
	}{
		{
			description: "artifact exists remotely",
			cache: &Cache{
				artifactCache: ArtifactCache{"hash": ImageDetails{
					Digest: "sha256@digest",
				}},
			},
			targetImageExistsRemotely: true,
			hashes:                    map[string]string{"image": "hash"},
			artifacts:                 []*latest.Artifact{{ImageName: "image"}},
			expectedBuildResults: []build.Artifact{
				{
					ImageName: "image",
					Tag:       "image:hash",
				},
			},
		}, {
			description: "artifact exists locally",
			cache: &Cache{
				artifactCache: ArtifactCache{"hash": ImageDetails{
					Digest: "sha256@digest",
				}},
				imageList: []types.ImageSummary{
					{
						RepoDigests: []string{"sha256@digest"},
						RepoTags:    []string{"image:hash"},
					},
				},
			},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image:hash": "image1:tag"},
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{"sha256@digest"},
						RepoTags:    []string{"image:hash"},
					},
				},
			},
			hashes:    map[string]string{"image": "hash"},
			artifacts: []*latest.Artifact{{ImageName: "image"}},
			expectedBuildResults: []build.Artifact{
				{
					ImageName: "image",
					Tag:       "image:hash",
				},
			},
			expectedPushedImages: []string{"image:hash"},
		}, {
			description: "artifact exists locally under a different tag",
			cache: &Cache{
				artifactCache: ArtifactCache{"hash": ImageDetails{
					ID: "imageID",
				}},
				imageList: []types.ImageSummary{{
					ID:       "imageID",
					RepoTags: []string{"image:anothertag"},
				}}},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image:anothertag": "imageID"},
			},
			hashes:    map[string]string{"image": "hash"},
			artifacts: []*latest.Artifact{{ImageName: "image"}},
			expectedBuildResults: []build.Artifact{
				{
					ImageName: "image",
					Tag:       "image:hash",
				},
			},
			expectedTaggedImages: []string{"image:hash"},
			expectedPushedImages: []string{"image:hash"},
		}, {
			description: "artifact doesn't exist",
			cache: &Cache{
				artifactCache: ArtifactCache{"hash": ImageDetails{
					ID: "imageID",
				}},
				imageList: []types.ImageSummary{{
					ID:       "imageID",
					RepoTags: []string{"image:anothertag"},
				}}},
			api:                      testutil.FakeAPIClient{},
			hashes:                   map[string]string{"image": "hash"},
			artifacts:                []*latest.Artifact{{ImageName: "image"}},
			expectedArtifactsToBuild: []*latest.Artifact{{ImageName: "image", WorkspaceHash: "hash"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			test.cache.useCache = true
			t.Override(&hashForArtifact, mockHashForArtifact(test.hashes))
			t.Override(&imgExistsRemotely, func(string, string, map[string]bool) bool {
				return test.targetImageExistsRemotely
			})

			test.cache.client = docker.NewLocalDaemon(&test.api, nil, false, map[string]bool{})
			actualArtifactsToBuild, actualBuildResults, err := test.cache.RetrieveCachedArtifacts(context.Background(), os.Stdout, test.artifacts)
			t.CheckError(false, err)
			t.CheckDeepEqual(actualArtifactsToBuild, test.expectedArtifactsToBuild)
			t.CheckDeepEqual(actualBuildResults, test.expectedBuildResults)
			t.CheckDeepEqual(test.expectedTaggedImages, test.api.Tagged)
			t.CheckDeepEqual(test.expectedPushedImages, test.api.PushedImages)
		})
	}
}

func TestRetrievePrebuiltImage(t *testing.T) {
	tests := []struct {
		description  string
		cache        *Cache
		imageDetails ImageDetails
		shouldErr    bool
		expected     string
	}{
		{
			description: "one image id exists",
			cache: &Cache{
				imageList: []types.ImageSummary{
					{
						RepoTags:    []string{"image:mytag"},
						RepoDigests: []string{image},
					},
					{
						RepoTags:    []string{"image1:latest"},
						RepoDigests: []string{imageOne},
					},
				},
			},
			imageDetails: ImageDetails{
				Digest: digest,
			},
			expected: "image:mytag",
		},
		{
			description: "no image id exists",
			cache: &Cache{
				imageList: []types.ImageSummary{
					{
						RepoTags:    []string{"image:mytag"},
						RepoDigests: []string{image},
					},
					{
						RepoTags:    []string{"image:mytag"},
						RepoDigests: []string{image},
					},
				},
			},
			shouldErr: true,
			imageDetails: ImageDetails{
				Digest: "dne",
			},
			expected: "",
		},
		{
			description: "one image id exists",
			cache: &Cache{
				imageList: []types.ImageSummary{
					{
						RepoTags: []string{"image1", "image2"},
						ID:       "something",
					},
					{
						RepoTags: []string{"image3"},
						ID:       "imageid",
					},
				},
			},
			imageDetails: ImageDetails{
				ID: "imageid",
			},
			expected: "image3",
		},
		{
			description: "multiple image ids exist",
			cache: &Cache{
				imageList: []types.ImageSummary{
					{
						RepoTags: []string{"image1", "image2"},
						ID:       "something",
					},
					{
						RepoTags: []string{"image3", "image4"},
						ID:       "imageid",
					},
				},
			},
			imageDetails: ImageDetails{
				ID: "imageid",
			},
			expected: "image3",
		},
		{
			description: "no image id exists",
			cache: &Cache{
				imageList: []types.ImageSummary{
					{
						RepoTags: []string{"image1", "image2"},
						ID:       "something",
					},
					{
						RepoTags: []string{"image3"},
						ID:       "somethingelse",
					},
				},
			},
			imageDetails: ImageDetails{
				ID: "imageid",
			},
			shouldErr: true,
			expected:  "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			test.cache.client = docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, map[string]bool{})

			actual, err := test.cache.retrievePrebuiltImage(test.imageDetails)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}
