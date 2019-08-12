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
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLookupLocal(t *testing.T) {
	tests := []struct {
		description string
		cache       map[string]ImageDetails
		api         *testutil.FakeAPIClient
		expected    cacheDetails
	}{
		{
			description: "miss",
			expected:    needsBuilding{hash: "hash"},
		},
		{
			description: "miss no imageID",
			cache: map[string]ImageDetails{
				"hash": {Digest: "ignored"},
			},
			expected: needsBuilding{hash: "hash"},
		},
		{
			description: "hit but not found",
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      &testutil.FakeAPIClient{},
			expected: needsBuilding{hash: "hash"},
		},
		{
			description: "hit but not found with error",
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api: &testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			expected: failed{err: errors.New("getting imageID for tag: inspecting image: ")},
		},
		{
			description: "hit",
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "imageID"),
			expected: found{hash: "hash"},
		},
		{
			description: "hit but different tag",
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "otherImageID").Add("othertag", "imageID"),
			expected: needsLocalTagging{hash: "hash", tag: "tag", imageID: "imageID"},
		},
		{
			description: "hit but imageID not found",
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "otherImageID"),
			expected: needsBuilding{hash: "hash"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&hashForArtifact, func(context.Context, DependencyLister, *latest.Artifact) (string, error) {
				return "hash", nil
			})
			t.Override(&buildInProgress, func(_ string) {})

			cache := &cache{
				imagesAreLocal: true,
				artifactCache:  test.cache,
				client:         docker.NewLocalDaemon(test.api, nil, false, nil),
			}
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
		cache       map[string]ImageDetails
		api         *testutil.FakeAPIClient
		expected    cacheDetails
	}{
		{
			description: "miss",
			expected:    needsBuilding{hash: "hash"},
		},
		{
			description: "hit",
			cache: map[string]ImageDetails{
				"hash": {Digest: "digest"},
			},
			expected: found{hash: "hash"},
		},
		{
			description: "hit with different tag",
			cache: map[string]ImageDetails{
				"hash": {Digest: "otherdigest"},
			},
			expected: needsRemoteTagging{hash: "hash", tag: "tag", digest: "otherdigest"},
		},
		{
			description: "found locally",
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      (&testutil.FakeAPIClient{}).Add("tag", "imageID"),
			expected: needsPushing{hash: "hash", tag: "tag", imageID: "imageID"},
		},
		{
			description: "not found",
			cache: map[string]ImageDetails{
				"hash": {ID: "imageID"},
			},
			api:      &testutil.FakeAPIClient{},
			expected: needsBuilding{hash: "hash"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&hashForArtifact, func(context.Context, DependencyLister, *latest.Artifact) (string, error) {
				return "hash", nil
			})
			t.Override(&docker.RemoteDigest, func(identifier string, _ map[string]bool) (string, error) {
				switch {
				case identifier == "tag":
					return "digest", nil
				case identifier == "tag@otherdigest":
					return "otherdigest", nil
				default:
					return "", errors.New("unknown remote tag")
				}
			})
			t.Override(&buildInProgress, func(_ string) {})

			cache := &cache{
				imagesAreLocal: false,
				artifactCache:  test.cache,
				client:         docker.NewLocalDaemon(test.api, nil, false, nil),
			}
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
