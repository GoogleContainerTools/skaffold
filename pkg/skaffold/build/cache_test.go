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

package build

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
	yaml "gopkg.in/yaml.v2"
)

var (
	digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
)

var defaultArtifactCache = ArtifactCache{"hash": ImageDetails{
	Digest: "digest",
	ID:     "id",
}}

func mockHashForArtifact(hashes map[string]string) func(context.Context, Builder, *latest.Artifact) (string, error) {
	return func(ctx context.Context, _ Builder, a *latest.Artifact) (string, error) {
		return hashes[a.ImageName], nil
	}
}

func Test_NewCache(t *testing.T) {
	client, err := docker.NewAPIClient()
	if err != nil {
		t.Fatalf("error gettting docker api client: %v", err)
	}
	tests := []struct {
		updateCacheFile   bool
		needsPush         bool
		name              string
		opts              *config.SkaffoldOptions
		expectedCache     *Cache
		cacheFileContents interface{}
	}{
		{
			name:              "get a valid cache from file",
			cacheFileContents: defaultArtifactCache,
			updateCacheFile:   true,
			opts: &config.SkaffoldOptions{
				CacheArtifacts: true,
			},
			expectedCache: &Cache{
				artifactCache: defaultArtifactCache,
				useCache:      true,
				client:        client,
			},
		},
		{
			name:              "needs push",
			cacheFileContents: defaultArtifactCache,
			needsPush:         true,
			updateCacheFile:   true,
			opts: &config.SkaffoldOptions{
				CacheArtifacts: true,
			},
			expectedCache: &Cache{
				artifactCache: defaultArtifactCache,
				useCache:      true,
				client:        client,
				needsPush:     true,
			},
		},
		{
			name:              "valid cache file exists, but useCache is false",
			cacheFileContents: defaultArtifactCache,
			opts:              &config.SkaffoldOptions{},
			expectedCache:     &Cache{},
		},
		{

			name:              "corrupted cache file",
			cacheFileContents: "corrupted cache file",
			opts: &config.SkaffoldOptions{
				CacheArtifacts: true,
			},
			expectedCache: &Cache{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			cacheFile := createTempCacheFile(t, test.cacheFileContents)

			if test.updateCacheFile {
				test.expectedCache.cacheFile = cacheFile
			}
			test.opts.CacheFile = cacheFile
			actualCache := NewCache(nil, test.opts, test.needsPush)

			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expectedCache, actualCache) {
				t.Errorf("Expected result different from actual result. Expected: %v, Actual: %v", test.expectedCache, actualCache)
			}
		})
	}
}

func Test_RetrieveCachedArtifacts(t *testing.T) {
	tests := []struct {
		name                 string
		cache                *Cache
		hashes               map[string]string
		artifacts            []*latest.Artifact
		expectedArtifacts    []*latest.Artifact
		api                  testutil.FakeAPIClient
		expectedBuildResults []Artifact
	}{
		{
			name:              "useCache is false, return all artifacts",
			cache:             &Cache{},
			artifacts:         []*latest.Artifact{{ImageName: "image1"}},
			expectedArtifacts: []*latest.Artifact{{ImageName: "image1"}},
		},
		{
			name:              "no artifacts in cache",
			cache:             &Cache{useCache: true},
			hashes:            map[string]string{"image1": "hash", "image2": "hash2"},
			artifacts:         []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			expectedArtifacts: []*latest.Artifact{{ImageName: "image1", WorkspaceHash: "hash"}, {ImageName: "image2", WorkspaceHash: "hash2"}},
		},
		{
			name: "one artifact in cache",
			cache: &Cache{
				useCache: true,
				artifactCache: ArtifactCache{"workspace-hash": ImageDetails{
					Digest: "sha256@digest",
				}},
			},
			hashes: map[string]string{"image1": "workspace-hash", "image2": "workspace-hash-2"},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image1:workspace-hash": "image1:tag"},
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{"sha256@digest"},
						RepoTags:    []string{"image1:workspace-hash"},
					},
				},
			},
			artifacts:            []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			expectedBuildResults: []Artifact{{ImageName: "image1", Tag: "image1:workspace-hash"}},
			expectedArtifacts:    []*latest.Artifact{{ImageName: "image2", WorkspaceHash: "workspace-hash-2"}},
		},
		{
			name: "both artifacts in cache, but only one exists locally",
			cache: &Cache{
				useCache: true,
				artifactCache: ArtifactCache{
					"hash":  ImageDetails{Digest: "sha256@digest1"},
					"hash2": ImageDetails{Digest: "sha256@digest2"},
				},
			},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image1:hash": "image1:tag"},
				ImageSummaries: []types.ImageSummary{
					{
						ID:          "id",
						RepoDigests: []string{"sha256@digest1"},
						RepoTags:    []string{"image1:hash"},
					},
				},
			},
			hashes:               map[string]string{"image1": "hash", "image2": "hash2"},
			artifacts:            []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			expectedArtifacts:    []*latest.Artifact{{ImageName: "image2", WorkspaceHash: "hash2"}},
			expectedBuildResults: []Artifact{{ImageName: "image1", Tag: "image1:hash"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalHash := hashForArtifact
			hashForArtifact = mockHashForArtifact(test.hashes)
			defer func() {
				hashForArtifact = originalHash
			}()

			originalLocal := localCluster
			localCluster = func() (bool, error) {
				return true, nil
			}
			defer func() {
				localCluster = originalLocal
			}()

			test.cache.client = docker.NewLocalDaemon(&test.api, nil)

			actualArtifacts, actualBuildResults := test.cache.RetrieveCachedArtifacts(context.Background(), os.Stdout, test.artifacts)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedArtifacts, actualArtifacts)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedBuildResults, actualBuildResults)
		})
	}
}

func createTempCacheFile(t *testing.T, cacheFileContents interface{}) string {
	temp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("error creating temp cache file: %v", err)
	}
	defer temp.Close()
	contents, err := yaml.Marshal(cacheFileContents)
	if err != nil {
		t.Fatalf("error marshalling cache: %v", err)
	}
	if err := ioutil.WriteFile(temp.Name(), contents, 0755); err != nil {
		t.Fatalf("error writing contents to %s: %v", temp.Name(), err)
	}
	return temp.Name()
}

func TestRetrieveCachedArtifactDetails(t *testing.T) {
	tests := []struct {
		name         string
		localCluster bool
		artifact     *latest.Artifact
		hashes       map[string]string
		digest       string
		api          *testutil.FakeAPIClient
		cache        *Cache
		expected     *cachedArtifactDetails
	}{
		{
			name:     "image doesn't exist in cache, remote cluster",
			artifact: &latest.Artifact{ImageName: "image"},
			hashes:   map[string]string{"image": "hash"},
			cache:    noCache,
			expected: &cachedArtifactDetails{
				needsRebuild: true,
			},
		},
		{
			name:         "image doesn't exist in cache, local cluster",
			artifact:     &latest.Artifact{ImageName: "image"},
			hashes:       map[string]string{"image": "hash"},
			localCluster: true,
			cache:        noCache,
			expected: &cachedArtifactDetails{
				needsRebuild: true,
			},
		},
		{
			name:     "image in cache and exists remotely, remote cluster",
			artifact: &latest.Artifact{ImageName: "image"},
			hashes:   map[string]string{"image": "hash"},
			api: &testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image:hash": "image:tag"},
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{"digest"},
						RepoTags:    []string{"image:hash"},
					},
				},
			},
			cache: &Cache{
				useCache:      true,
				artifactCache: ArtifactCache{"hash": ImageDetails{Digest: "digest"}},
			},
			digest: "digest",
			expected: &cachedArtifactDetails{
				hashTag: "image:hash",
			},
		},
		{
			name:         "image in cache and exists in daemon, local cluster",
			artifact:     &latest.Artifact{ImageName: "image"},
			hashes:       map[string]string{"image": "hash"},
			localCluster: true,
			api: &testutil.FakeAPIClient{
				TagToImageID: map[string]string{"image:hash": "image:tag"},
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{"digest"},
						RepoTags:    []string{"image:hash"},
					},
				},
			},
			cache: &Cache{
				useCache:      true,
				artifactCache: ArtifactCache{"hash": ImageDetails{Digest: "digest"}},
			},
			digest: "digest",
			expected: &cachedArtifactDetails{
				hashTag: "image:hash",
			},
		},
		{
			name:     "image in cache, prebuilt image exists, remote cluster",
			artifact: &latest.Artifact{ImageName: "image"},
			hashes:   map[string]string{"image": "hash"},
			api: &testutil.FakeAPIClient{
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{fmt.Sprintf("image@%s", digest)},
						RepoTags:    []string{"anotherimage:hash"},
					},
				},
			},
			cache: &Cache{
				useCache:      true,
				artifactCache: ArtifactCache{"hash": ImageDetails{Digest: digest}},
			},
			digest: digest,
			expected: &cachedArtifactDetails{
				needsRetag:    true,
				needsPush:     true,
				prebuiltImage: "anotherimage:hash",
				hashTag:       "image:hash",
			},
		},
		{
			name:         "image in cache, prebuilt image exists, local cluster",
			artifact:     &latest.Artifact{ImageName: "image"},
			hashes:       map[string]string{"image": "hash"},
			localCluster: true,
			api: &testutil.FakeAPIClient{
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{fmt.Sprintf("image@%s", digest)},
						RepoTags:    []string{"anotherimage:hash"},
					},
				},
			},
			cache: &Cache{
				useCache:      true,
				artifactCache: ArtifactCache{"hash": ImageDetails{Digest: digest}},
			},
			digest: digest,
			expected: &cachedArtifactDetails{
				needsRetag:    true,
				prebuiltImage: "anotherimage:hash",
				hashTag:       "image:hash",
			},
		},
		{
			name:         "needs push is true, local cluster",
			artifact:     &latest.Artifact{ImageName: "image"},
			hashes:       map[string]string{"image": "hash"},
			localCluster: true,
			api: &testutil.FakeAPIClient{
				ImageSummaries: []types.ImageSummary{
					{
						RepoDigests: []string{fmt.Sprintf("image@%s", digest)},
						RepoTags:    []string{"anotherimage:hash"},
					},
				},
			},
			cache: &Cache{
				useCache:      true,
				needsPush:     true,
				artifactCache: ArtifactCache{"hash": ImageDetails{Digest: digest}},
			},
			digest: digest,
			expected: &cachedArtifactDetails{
				needsRetag:    true,
				needsPush:     true,
				prebuiltImage: "anotherimage:hash",
				hashTag:       "image:hash",
			},
		},
		{
			name:     "no local daemon, image exists remotely",
			artifact: &latest.Artifact{ImageName: "image"},
			hashes:   map[string]string{"image": "hash"},
			cache: &Cache{
				useCache:      true,
				needsPush:     true,
				artifactCache: ArtifactCache{"hash": ImageDetails{Digest: digest}},
			},
			digest: digest,
			expected: &cachedArtifactDetails{
				hashTag: "image:hash",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalHash := hashForArtifact
			hashForArtifact = mockHashForArtifact(test.hashes)
			defer func() {
				hashForArtifact = originalHash
			}()

			originalLocal := localCluster
			localCluster = func() (bool, error) {
				return test.localCluster, nil
			}
			defer func() {
				localCluster = originalLocal
			}()

			originalRemoteDigest := remoteDigest
			remoteDigest = func(string) (string, error) {
				return test.digest, nil
			}
			defer func() {
				remoteDigest = originalRemoteDigest
			}()

			if test.api != nil {
				test.cache.client = docker.NewLocalDaemon(test.api, nil)
			}
			actual, err := test.cache.retrieveCachedArtifactDetails(context.Background(), test.artifact)
			if err != nil {
				t.Fatalf("error retrieving artifact details: %v", err)
			}
			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expected, actual) {
				t.Errorf("Expected: %v, Actual: %v", test.expected, actual)
			}
		})
	}
}

type mockBuilder struct {
	dependencies []string
}

func (m *mockBuilder) Labels() map[string]string { return nil }

func (m *mockBuilder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error) {
	return nil, nil
}

func (m *mockBuilder) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	return m.dependencies, nil
}

var mockCacheHasher = func(s string) (string, error) {
	return s, nil
}

func TestGetHashForArtifact(t *testing.T) {
	tests := []struct {
		name         string
		dependencies [][]string
		expected     string
	}{
		{
			name: "check dependencies in different orders",
			dependencies: [][]string{
				{"a", "b"},
				{"b", "a"},
			},
			expected: "eb394fd4559b1d9c383f4359667a508a615b82a74e1b160fce539f86ae0842e8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalHasher := cacheHasher
			hashFunction = mockCacheHasher
			defer func() {
				hashFunction = originalHasher
			}()

			for _, d := range test.dependencies {
				builder := &mockBuilder{dependencies: d}
				actual, err := getHashForArtifact(context.Background(), builder, nil)
				testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, actual)
			}
		})
	}
}

func TestCacheHasher(t *testing.T) {
	tests := []struct {
		name          string
		differentHash bool
		newFilename   string
		update        func(oldFile string, folder *testutil.TempDir)
	}{
		{
			name:          "change filename",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
			},
		},
		{
			name:          "change file contents",
			differentHash: true,
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			name:          "change both",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			name:          "change nothing",
			differentHash: false,
			update:        func(oldFile string, folder *testutil.TempDir) {},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalFile := "foo"
			originalContents := "contents"

			folder, cleanup := testutil.NewTempDir(t)
			defer cleanup()
			folder.Write(originalFile, originalContents)

			path := originalFile
			builder := &mockBuilder{dependencies: []string{folder.Path(originalFile)}}

			oldHash, err := getHashForArtifact(context.Background(), builder, nil)
			if err != nil {
				t.Errorf("error getting hash for artifact: %v", err)
			}

			test.update(originalFile, folder)
			if test.newFilename != "" {
				path = test.newFilename
			}

			builder.dependencies = []string{folder.Path(path)}
			newHash, err := getHashForArtifact(context.Background(), builder, nil)
			if err != nil {
				t.Errorf("error getting hash for artifact: %v", err)
			}

			if test.differentHash && oldHash == newHash {
				t.Fatalf("expected hashes to be different but they were the same:\n oldHash: %s\n newHash: %s", oldHash, newHash)
			}
			if !test.differentHash && oldHash != newHash {
				t.Fatalf("expected hashes to be the same but they were different:\n oldHash: %s\n newHash: %s", oldHash, newHash)
			}
		})
	}
}
