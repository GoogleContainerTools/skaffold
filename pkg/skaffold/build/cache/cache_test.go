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
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
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
		updateCacheFile    bool
		needsPush          bool
		updateClient       bool
		name               string
		opts               *config.SkaffoldOptions
		expectedCache      *Cache
		api                *testutil.FakeAPIClient
		cacheFileContents  interface{}
		insecureRegistries map[string]bool
	}{
		{
			name:              "get a valid cache from file",
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
				insecureRegistries: map[string]bool{
					"foo": true,
					"bar": true,
				},
			},
		},
		{
			name:              "needs push",
			cacheFileContents: defaultArtifactCache,
			needsPush:         true,
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
				needsPush:          true,
				insecureRegistries: emptyMap,
			},
		},
		{
			name:              "valid cache file exists, but useCache is false",
			cacheFileContents: defaultArtifactCache,
			api:               &testutil.FakeAPIClient{},
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
			restore := testutil.SetupFakeKubernetesContext(t, api.Config{CurrentContext: "cluster1"})
			defer restore()

			cacheFile := createTempCacheFile(t, test.cacheFileContents)
			if test.updateCacheFile {
				test.expectedCache.cacheFile = cacheFile
			}
			test.opts.CacheFile = cacheFile

			originalDockerClient := newDockerClient
			newDockerClient = func(map[string]bool) (docker.LocalDaemon, error) {
				return docker.NewLocalDaemon(test.api, nil, test.insecureRegistries), nil
			}
			defer func() {
				newDockerClient = originalDockerClient
			}()

			if test.updateClient {
				test.expectedCache.client = docker.NewLocalDaemon(test.api, nil, test.insecureRegistries)
			}

			actualCache := NewCache(context.Background(), nil, test.opts, test.needsPush, test.insecureRegistries)

			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expectedCache, actualCache) {
				t.Errorf("Expected result different from actual result. Expected: %v, Actual: %v", test.expectedCache, actualCache)
			}
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
