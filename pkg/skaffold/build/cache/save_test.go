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

func TestRetagLocalImages(t *testing.T) {
	tests := []struct {
		description      string
		api              *testutil.FakeAPIClient
		cache            *Cache
		artifactsToBuild []*latest.Artifact
		buildArtifacts   []build.Artifact
		expectedPush     []string
	}{
		{
			description: "retag and repush local image",
			cache: &Cache{
				useCache:       true,
				isLocalBuilder: true,
			},
			api: &testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image:tag": "imageid"},
				ImageSummaries: []types.ImageSummary{
					{
						ID:       "imageid",
						RepoTags: []string{"image:tag"},
					},
				},
			},
			artifactsToBuild: []*latest.Artifact{
				{
					ImageName:     "image",
					WorkspaceHash: "hash",
				},
			},
			buildArtifacts: []build.Artifact{
				{
					ImageName: "image",
					Tag:       "image:tag",
				},
			},
			expectedPush: []string{"image:hash"},
		}, {
			description: "build images remotely",
			api:         &testutil.FakeAPIClient{},
			cache: &Cache{
				useCache: true,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			test.cache.client = docker.NewLocalDaemon(test.api, nil, false, map[string]bool{})

			test.cache.RetagLocalImages(context.Background(), os.Stdout, test.artifactsToBuild, test.buildArtifacts)

			t.CheckDeepEqual(test.expectedPush, test.api.PushedImages)
		})
	}
}
