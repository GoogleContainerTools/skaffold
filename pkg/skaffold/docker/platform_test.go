/*
Copyright 2021 The Skaffold Authors

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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	spec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetPlatformsForImage(t *testing.T) {
	idx, err := random.Image(1024, 1)
	testutil.CheckError(t, false, err)
	expectedRepo := "foo/bar"
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)
	manifest, err := idx.Manifest()
	testutil.CheckError(t, false, err)

	configFile, err := idx.ConfigFile()
	testutil.CheckError(t, false, err)
	// Update image platform to match test
	configFile.OS = "linux"
	configFile.Architecture = "arm64"
	rawConfigFile, err := json.Marshal(configFile)
	testutil.CheckError(t, false, err)
	manifest.Config.Data = rawConfigFile
	cfgHash, cfgSize, err := v1.SHA256(bytes.NewReader(rawConfigFile))
	testutil.CheckError(t, false, err)
	manifest.Config.Digest = cfgHash
	manifest.Config.Size = cfgSize

	rawManifest, err := json.Marshal(manifest)
	testutil.CheckError(t, false, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case manifestPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			mt, err := idx.MediaType()
			testutil.CheckError(t, false, err)
			w.Header().Set("Content-Type", string(mt))
			w.Write(rawManifest)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	testutil.CheckError(t, false, err)
	tag := fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo)
	platforms, err := GetPlatforms(tag)
	testutil.CheckError(t, false, err)

	expectedPlatforms := []spec.Platform{
		{Architecture: "arm64", OS: "linux"},
	}
	testutil.CheckDeepEqual(t, expectedPlatforms, platforms)
}

func TestGetPlatformsForIndex(t *testing.T) {
	idx, err := random.Index(1024, 1, 3)
	testutil.CheckError(t, false, err)
	expectedRepo := "foo/bar"
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)
	manifest, err := idx.IndexManifest()
	testutil.CheckError(t, false, err)

	// Update image platform to match test
	manifest.Manifests[0].Platform = &v1.Platform{Architecture: "arm64", OS: "linux"}
	manifest.Manifests[1].Platform = &v1.Platform{Architecture: "amd64", OS: "linux"}
	rawManifest, err := json.Marshal(manifest)
	testutil.CheckError(t, false, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case manifestPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			mt, err := idx.MediaType()
			testutil.CheckError(t, false, err)
			w.Header().Set("Content-Type", string(mt))
			w.Write(rawManifest)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	testutil.CheckError(t, false, err)
	tag := fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo)
	platforms, err := GetPlatforms(tag)
	testutil.CheckError(t, false, err)

	expectedPlatforms := []spec.Platform{
		{Architecture: "arm64", OS: "linux"},
		{Architecture: "amd64", OS: "linux"},
	}
	testutil.CheckDeepEqual(t, expectedPlatforms, platforms)
}
