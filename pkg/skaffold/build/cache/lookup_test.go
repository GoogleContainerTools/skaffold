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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLookupLocal(t *testing.T) {
	tests := []struct {
		description string
		hasher      artifactHasher
		cache       map[string]ImageDetails
		api         *testutil.FakeAPIClient
		expected    cacheDetails
	}{
		{
			description: "miss",
			hasher:      mockHasher{"thehash"},
			api:         &testutil.FakeAPIClient{},
			cache:       map[string]ImageDetails{},
			expected:    needsBuilding{hash: "thehash"},
		},
		{
			description: "hash failure",
			hasher:      failingHasher{errors.New("BUG")},
			expected:    failed{err: errors.New("getting hash for artifact \"artifact\": BUG")},
		},
		{
			description: "miss no imageID",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {Digest: "ignored"},
			},
			expected: needsBuilding{hash: "hash"},
		},
		{
			description: "hit but not found",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      &testutil.FakeAPIClient{},
			expected: needsBuilding{hash: "hash"},
		},
		{
			description: "hit but not found with error",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api: &testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			expected: failed{err: sErrors.NewError(
				fmt.Errorf("getting imageID for tag: "),
				proto.ActionableErr{
					Message: "getting imageID for tag: ",
					ErrCode: proto.StatusCode_BUILD_DOCKER_GET_DIGEST_ERR,
				})},
		},
		{
			description: "hit",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "imageID"),
			expected: found{hash: "hash"},
		},
		{
			description: "hit but different tag",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "otherImageID").Add("othertag", "imageID"),
			expected: needsLocalTagging{hash: "hash", tag: "tag", imageID: "imageID"},
		},
		{
			description: "hit but imageID not found",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "otherImageID"),
			expected: needsBuilding{hash: "hash"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cache := &cache{
				isLocalImage:       func(string) (bool, error) { return true, nil },
				importMissingImage: func(imageName string) (bool, error) { return false, nil },
				artifactCache:      test.cache,
				client:             fakeLocalDaemon(test.api),
				cfg:                &mockConfig{mode: config.RunModes.Build},
			}

			t.Override(&newArtifactHasherFunc, func(_ build.ArtifactGraph, _ DependencyLister, _ config.RunMode) artifactHasher { return test.hasher })
			details := cache.lookupArtifacts(context.Background(), map[string]string{"artifact": "tag"}, []*latest.Artifact{{
				ImageName: "artifact",
			}})

			// cmp.Diff cannot access unexported fields in *exec.Cmd, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expected, details[0]) {
				t.Errorf("Expected result different from actual result. Expected: \n%v, \nActual: \n%v", test.expected, details)
			}
		})
	}
}

func TestLookupRemote(t *testing.T) {
	tests := []struct {
		description string
		hasher      artifactHasher
		cache       map[string]ImageDetails
		api         *testutil.FakeAPIClient
		expected    cacheDetails
	}{
		{
			description: "miss",
			hasher:      mockHasher{"hash"},
			api:         &testutil.FakeAPIClient{ErrImagePull: true},
			cache:       map[string]ImageDetails{},
			expected:    needsBuilding{hash: "hash"},
		},
		{
			description: "hash failure",
			hasher:      failingHasher{errors.New("BUG")},
			expected:    failed{err: errors.New("getting hash for artifact \"artifact\": BUG")},
		},
		{
			description: "hit",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {Digest: "digest"},
			},
			expected: found{hash: "hash"},
		},
		{
			description: "hit with different tag",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {Digest: "otherdigest"},
			},
			expected: needsRemoteTagging{hash: "hash", tag: "tag", digest: "otherdigest"},
		},
		{
			description: "found locally",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "imageID"),
			expected: needsPushing{hash: "hash", tag: "tag", imageID: "imageID"},
		},
		{
			description: "not found",
			hasher:      mockHasher{"hash"},
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      &testutil.FakeAPIClient{},
			expected: needsBuilding{hash: "hash"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&docker.RemoteDigest, func(identifier string, _ docker.Config) (string, error) {
				switch {
				case identifier == "tag":
					return "digest", nil
				case identifier == "tag@otherdigest":
					return "otherdigest", nil
				default:
					return "", errors.New("unknown remote tag")
				}
			})

			cache := &cache{
				isLocalImage:       func(string) (bool, error) { return false, nil },
				importMissingImage: func(imageName string) (bool, error) { return false, nil },
				artifactCache:      test.cache,
				client:             fakeLocalDaemon(test.api),
				cfg:                &mockConfig{mode: config.RunModes.Build},
			}
			t.Override(&newArtifactHasherFunc, func(_ build.ArtifactGraph, _ DependencyLister, _ config.RunMode) artifactHasher { return test.hasher })
			details := cache.lookupArtifacts(context.Background(), map[string]string{"artifact": "tag"}, []*latest.Artifact{{
				ImageName: "artifact",
			}})

			// cmp.Diff cannot access unexported fields in *exec.Cmd, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expected, details[0]) {
				t.Errorf("Expected result different from actual result. Expected: \n%v, \nActual: \n%v", test.expected, details)
			}
		})
	}
}

type mockHasher struct {
	val string
}

func (m mockHasher) hash(context.Context, *latest.Artifact) (string, error) {
	return m.val, nil
}

type failingHasher struct {
	err error
}

func (f failingHasher) hash(context.Context, *latest.Artifact) (string, error) {
	return "", f.err
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, false, nil)
}
