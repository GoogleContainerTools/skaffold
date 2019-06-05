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
	"fmt"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	digest    = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	digestOne = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	image     = fmt.Sprintf("image@%s", digest)
	imageOne  = fmt.Sprintf("image1@%s", digestOne)
	emptyMap  = map[string]bool{}
)

var defaultArtifactCache = ArtifactCache{"hash": ImageDetails{
	Digest: "digest",
	ID:     "id",
}}

func mockHashForArtifact(hashes map[string]string) func(context.Context, build.Builder, *latest.Artifact) (string, error) {
	return func(ctx context.Context, _ build.Builder, a *latest.Artifact) (string, error) {
		return hashes[a.ImageName], nil
	}
}

func Test_NewCache(t *testing.T) {
	tests := []struct {
		description        string
		updateCacheFile    bool
		pushImages         bool
		updateClient       bool
		opts               *config.SkaffoldOptions
		expectedCache      *Cache
		api                *testutil.FakeAPIClient
		cacheFileContents  interface{}
		insecureRegistries map[string]bool
	}{
		{
			description:       "get a valid cache from file",
			cacheFileContents: defaultArtifactCache,
			updateCacheFile:   true,
			opts: &config.SkaffoldOptions{
				CacheArtifacts: true,
			},
			updateClient: true,
			api: &testutil.FakeAPIClient{
				ImageSummaries: []types.ImageSummary{
					{
						ID: "image",
					},
				},
			},
			insecureRegistries: map[string]bool{
				"foo": true,
				"bar": true,
			},
			expectedCache: &Cache{
				artifactCache: defaultArtifactCache,
				useCache:      true,
				imageList: []types.ImageSummary{
					{
						ID: "image",
					},
				},
				isLocalBuilder: true,
				insecureRegistries: map[string]bool{
					"foo": true,
					"bar": true,
				},
			},
		},
		{
			description:       "needs push",
			cacheFileContents: defaultArtifactCache,
			pushImages:        true,
			updateCacheFile:   true,
			updateClient:      true,
			opts: &config.SkaffoldOptions{
				CacheArtifacts: true,
			},
			api:                &testutil.FakeAPIClient{},
			insecureRegistries: emptyMap,
			expectedCache: &Cache{
				artifactCache:      defaultArtifactCache,
				useCache:           true,
				isLocalBuilder:     true,
				pushImages:         true,
				insecureRegistries: emptyMap,
			},
		},
		{
			description:       "valid cache file exists, but useCache is false",
			cacheFileContents: defaultArtifactCache,
			api:               &testutil.FakeAPIClient{},
			opts:              &config.SkaffoldOptions{},
			expectedCache:     &Cache{},
		},
		{

			description:       "corrupted cache file",
			cacheFileContents: "corrupted cache file",
			opts: &config.SkaffoldOptions{
				CacheArtifacts: true,
			},
			expectedCache: &Cache{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			cacheFile := createTempCacheFile(t, test.cacheFileContents)
			if test.updateCacheFile {
				test.expectedCache.cacheFile = cacheFile
			}
			test.opts.CacheFile = cacheFile

			t.Override(&newDockerClient, func(forceRemove bool, insecureRegistries map[string]bool) (docker.LocalDaemon, error) {
				return docker.NewLocalDaemon(test.api, nil, forceRemove, test.insecureRegistries), nil
			})

			if test.updateClient {
				test.expectedCache.client = docker.NewLocalDaemon(test.api, nil, false, test.insecureRegistries)
			}

			runCtx := &runcontext.RunContext{
				Opts: test.opts,
				Cfg: &latest.Pipeline{
					Build: latest.BuildConfig{
						BuildType: latest.BuildType{
							LocalBuild: &latest.LocalBuild{
								Push: &test.pushImages,
							},
						},
					},
				},
				InsecureRegistries: test.insecureRegistries,
			}

			actualCache := NewCache(nil, runCtx)

			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expectedCache, actualCache) {
				t.Errorf("Expected result different from actual result. Expected: %v, Actual: %v", test.expectedCache, actualCache)
			}
		})
	}
}

func createTempCacheFile(t *testutil.T, cacheFileContents interface{}) string {
	contents, err := yaml.Marshal(cacheFileContents)
	if err != nil {
		t.Fatalf("error marshalling cache: %v", err)
	}

	return t.TempFile("", contents)
}
