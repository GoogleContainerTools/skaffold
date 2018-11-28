// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const bogusDigest = "sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

func mustDigest(t *testing.T, img v1.Image) v1.Hash {
	h, err := img.Digest()
	if err != nil {
		t.Fatalf("Digest() = %v", err)
	}
	return h
}

func mustManifest(t *testing.T, img v1.Image) *v1.Manifest {
	m, err := img.Manifest()
	if err != nil {
		t.Fatalf("Manifest() = %v", err)
	}
	return m
}

func mustRawManifest(t *testing.T, img v1.Image) []byte {
	m, err := img.RawManifest()
	if err != nil {
		t.Fatalf("RawManifest() = %v", err)
	}
	return m
}

func mustRawConfigFile(t *testing.T, img v1.Image) []byte {
	c, err := img.RawConfigFile()
	if err != nil {
		t.Fatalf("RawConfigFile() = %v", err)
	}
	return c
}

func randomImage(t *testing.T) v1.Image {
	rnd, err := random.Image(1024, 1)
	if err != nil {
		t.Fatalf("random.Image() = %v", err)
	}
	return rnd
}

func newReference(host, repo, ref string) (name.Reference, error) {
	tag, err := name.NewTag(fmt.Sprintf("%s/%s:%s", host, repo, ref), name.WeakValidation)
	if err == nil {
		return tag, nil
	}
	return name.NewDigest(fmt.Sprintf("%s/%s@%s", host, repo, ref), name.WeakValidation)
}

// TODO(jonjohnsonjr): Make this real.
func TestMediaType(t *testing.T) {
	img := remoteImage{}
	got, err := img.MediaType()
	if err != nil {
		t.Fatalf("MediaType() = %v", err)
	}
	want := types.DockerManifestSchema2
	if got != want {
		t.Errorf("MediaType() = %v, want %v", got, want)
	}
}

func TestRawManifestDigests(t *testing.T) {
	img := randomImage(t)
	expectedRepo := "foo/bar"

	cases := []struct {
		name          string
		ref           string
		responseBody  []byte
		contentDigest string
		wantErr       bool
	}{{
		name:          "normal pull, by tag",
		ref:           "latest",
		responseBody:  mustRawManifest(t, img),
		contentDigest: mustDigest(t, img).String(),
		wantErr:       false,
	}, {
		name:          "normal pull, by digest",
		ref:           mustDigest(t, img).String(),
		responseBody:  mustRawManifest(t, img),
		contentDigest: mustDigest(t, img).String(),
		wantErr:       false,
	}, {
		name:          "right content-digest, wrong body, by digest",
		ref:           mustDigest(t, img).String(),
		responseBody:  []byte("not even json"),
		contentDigest: mustDigest(t, img).String(),
		wantErr:       true,
	}, {
		name:          "right body, wrong content-digest, by tag",
		ref:           "latest",
		responseBody:  mustRawManifest(t, img),
		contentDigest: bogusDigest,
		wantErr:       false,
	}, {
		// NB: This succeeds! We don't care what the registry thinks.
		name:          "right body, wrong content-digest, by digest",
		ref:           mustDigest(t, img).String(),
		responseBody:  mustRawManifest(t, img),
		contentDigest: bogusDigest,
		wantErr:       false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifestPath := fmt.Sprintf("/v2/%s/manifests/%s", expectedRepo, tc.ref)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case manifestPath:
					if r.Method != http.MethodGet {
						t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
					}

					w.Header().Set("Docker-Content-Digest", tc.contentDigest)
					w.Write(tc.responseBody)
				default:
					t.Fatalf("Unexpected path: %v", r.URL.Path)
				}
			}))
			defer server.Close()
			u, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("url.Parse(%v) = %v", server.URL, err)
			}

			ref, err := newReference(u.Host, expectedRepo, tc.ref)
			if err != nil {
				t.Fatalf("url.Parse(%v, %v, %v) = %v", u.Host, expectedRepo, tc.ref, err)
			}

			rmt := remoteImage{
				ref:    ref,
				client: http.DefaultClient,
			}

			if _, err := rmt.RawManifest(); (err != nil) != tc.wantErr {
				t.Errorf("RawManifest() wrong error: %v, want %v: %v\n", (err != nil), tc.wantErr, err)
			}
		})
	}
}

func TestRawManifestNotFound(t *testing.T) {
	expectedRepo := "foo/bar"
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case manifestPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%v) = %v", server.URL, err)
	}

	img := remoteImage{
		ref:    mustNewTag(t, fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo)),
		client: http.DefaultClient,
	}

	if _, err := img.RawManifest(); err == nil {
		t.Error("RawManifest() = nil; wanted error")
	}
}

func TestRawConfigFileNotFound(t *testing.T) {
	img := randomImage(t)
	expectedRepo := "foo/bar"
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)
	configPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, mustConfigName(t, img))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case configPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.WriteHeader(http.StatusNotFound)
		case manifestPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.Write(mustRawManifest(t, img))
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%v) = %v", server.URL, err)
	}

	rmt := remoteImage{
		ref:    mustNewTag(t, fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo)),
		client: http.DefaultClient,
	}

	if _, err := rmt.RawConfigFile(); err == nil {
		t.Error("RawConfigFile() = nil; wanted error")
	}
}

func TestAcceptHeaders(t *testing.T) {
	img := randomImage(t)
	expectedRepo := "foo/bar"
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case manifestPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			if got, want := r.Header.Get("Accept"), string(types.DockerManifestSchema2); got != want {
				t.Errorf("Accept header; got %v, want %v", got, want)
			}
			w.Write(mustRawManifest(t, img))
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%v) = %v", server.URL, err)
	}

	rmt := &remoteImage{
		ref:    mustNewTag(t, fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo)),
		client: http.DefaultClient,
	}
	manifest, err := rmt.RawManifest()
	if err != nil {
		t.Errorf("RawManifest() = %v", err)
	}
	if got, want := manifest, mustRawManifest(t, img); bytes.Compare(got, want) != 0 {
		t.Errorf("RawManifest() = %v, want %v", got, want)
	}
}

func TestImage(t *testing.T) {
	img := randomImage(t)
	expectedRepo := "foo/bar"
	layerDigest := mustManifest(t, img).Layers[0].Digest
	layerSize := mustManifest(t, img).Layers[0].Size
	configPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, mustConfigName(t, img))
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)
	layerPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, layerDigest)
	manifestReqCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case configPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.Write(mustRawConfigFile(t, img))
		case manifestPath:
			manifestReqCount++
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.Write(mustRawManifest(t, img))
		case layerPath:
			t.Fatalf("BlobSize should not make any request: %v", r.URL.Path)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%v) = %v", server.URL, err)
	}

	tag := mustNewTag(t, fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo))
	rmt, err := Image(tag, WithTransport(http.DefaultTransport))
	if err != nil {
		t.Errorf("Image() = %v", err)
	}

	if got, want := mustRawManifest(t, rmt), mustRawManifest(t, img); bytes.Compare(got, want) != 0 {
		t.Errorf("RawManifest() = %v, want %v", got, want)
	}
	if got, want := mustRawConfigFile(t, rmt), mustRawConfigFile(t, img); bytes.Compare(got, want) != 0 {
		t.Errorf("RawConfigFile() = %v, want %v", got, want)
	}
	// Make sure caching the manifest works.
	if manifestReqCount != 1 {
		t.Errorf("RawManifest made %v requests, expected 1", manifestReqCount)
	}

	l, err := rmt.LayerByDigest(layerDigest)
	if err != nil {
		t.Errorf("LayerByDigest() = %v", err)
	}
	// BlobSize should not HEAD.
	size, err := l.Size()
	if err != nil {
		t.Errorf("BlobSize() = %v", err)
	}
	if got, want := size, layerSize; want != got {
		t.Errorf("BlobSize() = %v want %v", got, want)
	}
}
