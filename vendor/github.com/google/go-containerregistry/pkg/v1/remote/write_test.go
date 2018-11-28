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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/stream"
)

func mustNewTag(t *testing.T, s string) name.Tag {
	tag, err := name.NewTag(s, name.WeakValidation)
	if err != nil {
		t.Fatalf("NewTag(%v) = %v", s, err)
	}
	return tag
}

func TestUrl(t *testing.T) {
	tests := []struct {
		tag  string
		path string
		url  string
	}{{
		tag:  "gcr.io/foo/bar:latest",
		path: "/v2/foo/bar/manifests/latest",
		url:  "https://gcr.io/v2/foo/bar/manifests/latest",
	}, {
		tag:  "localhost:8080/foo/bar:baz",
		path: "/v2/foo/bar/blobs/upload",
		url:  "http://localhost:8080/v2/foo/bar/blobs/upload",
	}}

	for _, test := range tests {
		w := &writer{
			ref: mustNewTag(t, test.tag),
		}
		if got, want := w.url(test.path), test.url; got.String() != want {
			t.Errorf("url(%v) = %v, want %v", test.path, got.String(), want)
		}
	}
}

func TestNextLocation(t *testing.T) {
	tests := []struct {
		location string
		url      string
	}{{
		location: "https://gcr.io/v2/foo/bar/blobs/uploads/1234567?baz=blah",
		url:      "https://gcr.io/v2/foo/bar/blobs/uploads/1234567?baz=blah",
	}, {
		location: "/v2/foo/bar/blobs/uploads/1234567?baz=blah",
		url:      "https://gcr.io/v2/foo/bar/blobs/uploads/1234567?baz=blah",
	}}

	ref := mustNewTag(t, "gcr.io/foo/bar:latest")
	w := &writer{
		ref: ref,
	}

	for _, test := range tests {
		resp := &http.Response{
			Header: map[string][]string{
				"Location": {test.location},
			},
			Request: &http.Request{
				URL: &url.URL{
					Scheme: ref.Registry.Scheme(),
					Host:   ref.RegistryStr(),
				},
			},
		}

		got, err := w.nextLocation(resp)
		if err != nil {
			t.Errorf("nextLocation(%v) = %v", resp, err)
		}
		want := test.url
		if got != want {
			t.Errorf("nextLocation(%v) = %v, want %v", resp, got, want)
		}
	}
}

type closer interface {
	Close()
}

func setupImage(t *testing.T) v1.Image {
	rnd, err := random.Image(1024, 1)
	if err != nil {
		t.Fatalf("random.Image() = %v", err)
	}
	return rnd
}

func mustConfigName(t *testing.T, img v1.Image) v1.Hash {
	h, err := img.ConfigName()
	if err != nil {
		t.Fatalf("ConfigName() = %v", err)
	}
	return h
}

func setupWriter(repo string, img v1.Image, handler http.HandlerFunc) (*writer, closer, error) {
	server := httptest.NewServer(handler)
	return setupWriterWithServer(server, repo, img)
}

func setupWriterWithServer(server *httptest.Server, repo string, img v1.Image) (*writer, closer, error) {
	u, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		return nil, nil, err
	}
	tag, err := name.NewTag(fmt.Sprintf("%s/%s:latest", u.Host, repo), name.WeakValidation)
	if err != nil {
		server.Close()
		return nil, nil, err
	}

	return &writer{
		ref:    tag,
		img:    img,
		client: http.DefaultClient,
	}, server, nil
}

func TestCheckExistingFound(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedRepo := "foo/bar"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, h.String())

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodHead)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		http.Error(w, "Found", http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	existing, err := w.checkExisting(h)
	if err != nil {
		t.Errorf("checkExisting() = %v", err)
	}
	if !existing {
		t.Error("checkExisting() = !existing, want existing")
	}
}

func TestCheckExistingNotFound(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedRepo := "foo/bar"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, h.String())

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodHead)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		http.Error(w, "NotFound", http.StatusNotFound)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	existing, err := w.checkExisting(h)
	if err != nil {
		t.Errorf("checkExisting() = %v", err)
	}
	if existing {
		t.Error("checkExisting() = existing, want !existing")
	}
}

func TestCheckExistingError(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedRepo := "foo/bar"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, h.String())

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodHead)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		http.Error(w, "Found - Error", http.StatusBadRequest)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	existing, err := w.checkExisting(h)
	if err == nil {
		t.Errorf("checkExisting() = %v; wanted error", existing)
	}
}

func TestInitiateUploadNoMountsExists(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedRepo := "foo/bar"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	expectedQuery := url.Values{
		"mount": []string{h.String()},
	}.Encode()

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("RawQuery; got %v, want %v", r.URL.RawQuery, expectedQuery)
		}
		http.Error(w, "Mounted", http.StatusCreated)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	_, mounted, err := w.initiateUpload("", h.String())
	if err != nil {
		t.Errorf("intiateUpload() = %v", err)
	}
	if !mounted {
		t.Error("initiateUpload() = !mounted, want mounted")
	}
}

func TestInitiateUploadNoMountsInitiated(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedRepo := "baz/blah"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	expectedQuery := url.Values{
		"mount": []string{h.String()},
	}.Encode()
	expectedLocation := "https://somewhere.io/upload?foo=bar"

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("RawQuery; got %v, want %v", r.URL.RawQuery, expectedQuery)
		}
		w.Header().Set("Location", expectedLocation)
		http.Error(w, "Initiated", http.StatusAccepted)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	location, mounted, err := w.initiateUpload("", h.String())
	if err != nil {
		t.Errorf("intiateUpload() = %v", err)
	}
	if mounted {
		t.Error("initiateUpload() = mounted, want !mounted")
	}
	if location != expectedLocation {
		t.Errorf("initiateUpload(); got %v, want %v", location, expectedLocation)
	}
}

func TestInitiateUploadNoMountsBadStatus(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedRepo := "ugh/another"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	expectedQuery := url.Values{
		"mount": []string{h.String()},
	}.Encode()

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("RawQuery; got %v, want %v", r.URL.RawQuery, expectedQuery)
		}
		http.Error(w, "Unknown", http.StatusNoContent)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	location, mounted, err := w.initiateUpload("", h.String())
	if err == nil {
		t.Errorf("intiateUpload() = %v, %v; wanted error", location, mounted)
	}
}

func TestInitiateUploadMountsWithMountFromDifferentRegistry(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedMountRepo := "a/different/repo"
	expectedRepo := "yet/again"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	expectedQuery := url.Values{
		"mount": []string{h.String()},
	}.Encode()

	img = &mountableImage{
		Image:     img,
		Reference: mustNewTag(t, fmt.Sprintf("gcr.io/%s", expectedMountRepo)),
	}

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("RawQuery; got %v, want %v", r.URL.RawQuery, expectedQuery)
		}
		http.Error(w, "Mounted", http.StatusCreated)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	_, mounted, err := w.initiateUpload("", h.String())
	if err != nil {
		t.Errorf("intiateUpload() = %v", err)
	}
	if !mounted {
		t.Error("initiateUpload() = !mounted, want mounted")
	}
}

func TestInitiateUploadMountsWithMountFromTheSameRegistry(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedMountRepo := "a/different/repo"
	expectedRepo := "yet/again"
	expectedPath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	expectedQuery := url.Values{
		"mount": []string{h.String()},
		"from":  []string{expectedMountRepo},
	}.Encode()

	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("RawQuery; got %v, want %v", r.URL.RawQuery, expectedQuery)
		}
		http.Error(w, "Mounted", http.StatusCreated)
	})
	server := httptest.NewServer(serverHandler)
	u, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		t.Fatalf("httptest.NewServer() = %v", err)
	}

	img = &mountableImage{
		Image:     img,
		Reference: mustNewTag(t, fmt.Sprintf("%s/%s", u.Host, expectedMountRepo)),
	}

	w, closer, err := setupWriterWithServer(server, expectedRepo, img)
	if err != nil {
		t.Fatalf("setupWriterWithServer() = %v", err)
	}
	defer closer.Close()

	_, mounted, err := w.initiateUpload(expectedMountRepo, h.String())
	if err != nil {
		t.Errorf("intiateUpload() = %v", err)
	}
	if !mounted {
		t.Error("initiateUpload() = !mounted, want mounted")
	}
}

func TestStreamBlob(t *testing.T) {
	img := setupImage(t)
	expectedPath := "/vWhatever/I/decide"
	expectedCommitLocation := "https://commit.io/v12/blob"

	w, closer, err := setupWriter("what/ever", img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPatch)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		got, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll(Body) = %v", err)
		}
		want, err := img.RawConfigFile()
		if err != nil {
			t.Errorf("RawConfigFile() = %v", err)
		}
		if bytes.Compare(got, want) != 0 {
			t.Errorf("bytes.Compare(); got %v, want %v", got, want)
		}
		w.Header().Set("Location", expectedCommitLocation)
		http.Error(w, "Created", http.StatusCreated)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	streamLocation := w.url(expectedPath)

	l, err := partial.ConfigLayer(img)
	if err != nil {
		t.Fatalf("ConfigLayer: %v", err)
	}
	blob, err := l.Compressed()
	if err != nil {
		t.Fatalf("layer.Compressed: %v", err)
	}

	commitLocation, err := w.streamBlob(blob, streamLocation.String())
	if err != nil {
		t.Errorf("streamBlob() = %v", err)
	}
	if commitLocation != expectedCommitLocation {
		t.Errorf("streamBlob(); got %v, want %v", commitLocation, expectedCommitLocation)
	}
}

func TestStreamLayer(t *testing.T) {
	var n, wantSize int64 = 10000, 49
	newBlob := func() io.ReadCloser { return ioutil.NopCloser(bytes.NewReader(bytes.Repeat([]byte{'a'}, int(n)))) }
	wantDigest := "sha256:3d7c465be28d9e1ed810c42aeb0e747b44441424f566722ba635dc93c947f30e"

	expectedPath := "/vWhatever/I/decide"
	expectedCommitLocation := "https://commit.io/v12/blob"
	w, closer, err := setupWriter("what/ever", nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPatch)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}

		h := sha256.New()
		s, err := io.Copy(h, r.Body)
		if err != nil {
			t.Errorf("Reading body: %v", err)
		}
		if s != wantSize {
			t.Errorf("Received %d bytes, want %d", s, wantSize)
		}
		gotDigest := "sha256:" + hex.EncodeToString(h.Sum(nil))
		if gotDigest != wantDigest {
			t.Errorf("Received bytes with digest %q, want %q", gotDigest, wantDigest)
		}

		w.Header().Set("Location", expectedCommitLocation)
		http.Error(w, "Created", http.StatusCreated)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	streamLocation := w.url(expectedPath)
	sl := stream.NewLayer(newBlob())
	blob, err := sl.Compressed()
	if err != nil {
		t.Fatalf("layer.Compressed: %v", err)
	}

	commitLocation, err := w.streamBlob(blob, streamLocation.String())
	if err != nil {
		t.Errorf("streamBlob: %v", err)
	}
	if commitLocation != expectedCommitLocation {
		t.Errorf("streamBlob(); got %v, want %v", commitLocation, expectedCommitLocation)
	}
}

func TestCommitBlob(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedPath := "/no/commitment/issues"
	expectedQuery := url.Values{
		"digest": []string{h.String()},
	}.Encode()

	w, closer, err := setupWriter("what/ever", img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPut)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("RawQuery; got %v, want %v", r.URL.RawQuery, expectedQuery)
		}
		http.Error(w, "Created", http.StatusCreated)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	commitLocation := w.url(expectedPath)

	if err := w.commitBlob(commitLocation.String(), h.String()); err != nil {
		t.Errorf("commitBlob() = %v", err)
	}
}

func TestUploadOne(t *testing.T) {
	img := setupImage(t)
	h := mustConfigName(t, img)
	expectedRepo := "baz/blah"
	headPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, h.String())
	initiatePath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	streamPath := "/path/to/upload"
	commitPath := "/path/to/commit"

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case headPath:
			if r.Method != http.MethodHead {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodHead)
			}
			http.Error(w, "NotFound", http.StatusNotFound)
		case initiatePath:
			if r.Method != http.MethodPost {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
			}
			w.Header().Set("Location", streamPath)
			http.Error(w, "Initiated", http.StatusAccepted)
		case streamPath:
			if r.Method != http.MethodPatch {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPatch)
			}
			got, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Errorf("ReadAll(Body) = %v", err)
			}
			want, err := img.RawConfigFile()
			if err != nil {
				t.Errorf("RawConfigFile() = %v", err)
			}
			if bytes.Compare(got, want) != 0 {
				t.Errorf("bytes.Compare(); got %v, want %v", got, want)
			}
			w.Header().Set("Location", commitPath)
			http.Error(w, "Initiated", http.StatusAccepted)
		case commitPath:
			if r.Method != http.MethodPut {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPut)
			}
			http.Error(w, "Created", http.StatusCreated)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	l, err := partial.ConfigLayer(img)
	if err != nil {
		t.Fatalf("ConfigLayer: %v", err)
	}
	if err := w.uploadOne(l); err != nil {
		t.Errorf("uploadOne() = %v", err)
	}
}

func TestUploadOneStreamedLayer(t *testing.T) {
	expectedRepo := "baz/blah"
	initiatePath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	streamPath := "/path/to/upload"
	commitPath := "/path/to/commit"

	w, closer, err := setupWriter(expectedRepo, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case initiatePath:
			if r.Method != http.MethodPost {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
			}
			w.Header().Set("Location", streamPath)
			http.Error(w, "Initiated", http.StatusAccepted)
		case streamPath:
			if r.Method != http.MethodPatch {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPatch)
			}
			// TODO(jasonhall): What should we check here?
			w.Header().Set("Location", commitPath)
			http.Error(w, "Initiated", http.StatusAccepted)
		case commitPath:
			if r.Method != http.MethodPut {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPut)
			}
			http.Error(w, "Created", http.StatusCreated)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	var n, wantSize int64 = 10000, 49
	newBlob := func() io.ReadCloser { return ioutil.NopCloser(bytes.NewReader(bytes.Repeat([]byte{'a'}, int(n)))) }
	wantDigest := "sha256:3d7c465be28d9e1ed810c42aeb0e747b44441424f566722ba635dc93c947f30e"
	wantDiffID := "sha256:27dd1f61b867b6a0f6e9d8a41c43231de52107e53ae424de8f847b821db4b711"
	l := stream.NewLayer(newBlob())
	if err := w.uploadOne(l); err != nil {
		t.Fatalf("uploadOne: %v", err)
	}

	if dig, err := l.Digest(); err != nil {
		t.Errorf("Digest: %v", err)
	} else if dig.String() != wantDigest {
		t.Errorf("Digest got %q, want %q", dig, wantDigest)
	}
	if diffID, err := l.DiffID(); err != nil {
		t.Errorf("DiffID: %v", err)
	} else if diffID.String() != wantDiffID {
		t.Errorf("DiffID got %q, want %q", diffID, wantDiffID)
	}
	if size, err := l.Size(); err != nil {
		t.Errorf("Size: %v", err)
	} else if size != wantSize {
		t.Errorf("Size got %d, want %d", size, wantSize)
	}
}

func TestCommitImage(t *testing.T) {
	img := setupImage(t)

	expectedRepo := "foo/bar"
	expectedPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)

	w, closer, err := setupWriter(expectedRepo, img, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Method; got %v, want %v", r.Method, http.MethodPut)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("URL; got %v, want %v", r.URL.Path, expectedPath)
		}
		got, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll(Body) = %v", err)
		}
		want, err := img.RawManifest()
		if err != nil {
			t.Errorf("RawManifest() = %v", err)
		}
		if bytes.Compare(got, want) != 0 {
			t.Errorf("bytes.Compare(); got %v, want %v", got, want)
		}
		mt, err := img.MediaType()
		if err != nil {
			t.Errorf("MediaType() = %v", err)
		}
		if got, want := r.Header.Get("Content-Type"), string(mt); got != want {
			t.Errorf("Header; got %v, want %v", got, want)
		}
		http.Error(w, "Created", http.StatusCreated)
	}))
	if err != nil {
		t.Fatalf("setupWriter() = %v", err)
	}
	defer closer.Close()

	if err := w.commitImage(); err != nil {
		t.Errorf("commitBlob() = %v", err)
	}
}

func TestWrite(t *testing.T) {
	img := setupImage(t)
	expectedRepo := "write/time"
	headPathPrefix := fmt.Sprintf("/v2/%s/blobs/", expectedRepo)
	initiatePath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && strings.HasPrefix(r.URL.Path, headPathPrefix) && r.URL.Path != initiatePath {
			http.Error(w, "NotFound", http.StatusNotFound)
			return
		}
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case initiatePath:
			if r.Method != http.MethodPost {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
			}
			http.Error(w, "Mounted", http.StatusCreated)
		case manifestPath:
			if r.Method != http.MethodPut {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPut)
			}
			http.Error(w, "Created", http.StatusCreated)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%v) = %v", server.URL, err)
	}
	tag, err := name.NewTag(fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo), name.WeakValidation)
	if err != nil {
		t.Fatalf("NewTag() = %v", err)
	}

	if err := Write(tag, img, authn.Anonymous, http.DefaultTransport); err != nil {
		t.Errorf("Write() = %v", err)
	}
}

func TestWriteWithErrors(t *testing.T) {
	img := setupImage(t)
	expectedRepo := "write/time"
	headPathPrefix := fmt.Sprintf("/v2/%s/blobs/", expectedRepo)
	initiatePath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)

	expectedError := &Error{
		Errors: []Diagnostic{{
			Code:    NameInvalidErrorCode,
			Message: "some explanation of how things were messed up.",
		}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && strings.HasPrefix(r.URL.Path, headPathPrefix) && r.URL.Path != initiatePath {
			http.Error(w, "NotFound", http.StatusNotFound)
			return
		}
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case initiatePath:
			if r.Method != http.MethodPost {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
			}
			b, err := json.Marshal(expectedError)
			if err != nil {
				t.Fatalf("json.Marshal() = %v", err)
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(b)
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%v) = %v", server.URL, err)
	}
	tag, err := name.NewTag(fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo), name.WeakValidation)
	if err != nil {
		t.Fatalf("NewTag() = %v", err)
	}

	if err := Write(tag, img, authn.Anonymous, http.DefaultTransport); err == nil {
		t.Error("Write() = nil; wanted error")
	} else if se, ok := err.(*Error); !ok {
		t.Errorf("Write() = %T; wanted *remote.Error", se)
	} else if diff := cmp.Diff(expectedError, se); diff != "" {
		t.Errorf("Write(); (-want +got) = %s", diff)
	}
}

func TestScopesForUploadingImage(t *testing.T) {
	referenceToUpload, err := name.NewTag("example.com/sample/sample:latest", name.WeakValidation)
	if err != nil {
		t.Fatalf("name.NewTag() = %v", err)
	}

	anotherRepo1, err := name.NewTag("example.com/sample/another_repo1:latest", name.WeakValidation)
	if err != nil {
		t.Fatalf("name.NewTag() = %v", err)
	}

	anotherRepo2, err := name.NewTag("example.com/sample/another_repo2:latest", name.WeakValidation)
	if err != nil {
		t.Fatalf("name.NewTag() = %v", err)
	}

	img := setupImage(t)
	layers, err := img.Layers()
	if err != nil {
		t.Fatalf("img.Layers() = %v", err)
	}
	dummyLayer := layers[0]

	testCases := []struct {
		name      string
		reference name.Reference
		layers    []v1.Layer
		expected  []string
	}{
		{
			name:      "empty layers",
			reference: referenceToUpload,
			layers:    []v1.Layer{},
			expected: []string{
				referenceToUpload.Scope(transport.PushScope),
			},
		},
		{
			name:      "mountable layers with single reference with no-duplicate",
			reference: referenceToUpload,
			layers: []v1.Layer{
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo1,
				},
			},
			expected: []string{
				referenceToUpload.Scope(transport.PushScope),
				anotherRepo1.Scope(transport.PullScope),
			},
		},
		{
			name:      "mountable layers with single reference with duplicate",
			reference: referenceToUpload,
			layers: []v1.Layer{
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo1,
				},
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo1,
				},
			},
			expected: []string{
				referenceToUpload.Scope(transport.PushScope),
				anotherRepo1.Scope(transport.PullScope),
			},
		},
		{
			name:      "mountable layers with multiple references with no-duplicates",
			reference: referenceToUpload,
			layers: []v1.Layer{
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo1,
				},
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo2,
				},
			},
			expected: []string{
				referenceToUpload.Scope(transport.PushScope),
				anotherRepo1.Scope(transport.PullScope),
				anotherRepo2.Scope(transport.PullScope),
			},
		},
		{
			name:      "mountable layers with multiple references with duplicates",
			reference: referenceToUpload,
			layers: []v1.Layer{
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo1,
				},
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo2,
				},
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo1,
				},
				&MountableLayer{
					Layer:     dummyLayer,
					Reference: anotherRepo2,
				},
			},
			expected: []string{
				referenceToUpload.Scope(transport.PushScope),
				anotherRepo1.Scope(transport.PullScope),
				anotherRepo2.Scope(transport.PullScope),
			},
		},
	}

	for _, tc := range testCases {
		actual := scopesForUploadingImage(tc.reference, tc.layers)

		if want, got := tc.expected[0], actual[0]; want != got {
			t.Errorf("TestScopesForUploadingImage() %s: Wrong first scope; want %v, got %v", tc.name, want, got)
		}

		less := func(a, b string) bool {
			return strings.Compare(a, b) <= -1
		}
		if diff := cmp.Diff(tc.expected[1:], actual[1:], cmpopts.SortSlices(less)); diff != "" {
			t.Errorf("TestScopesForUploadingImage() %s: Wrong scopes (-want +got) = %v", tc.name, diff)
		}
	}
}
