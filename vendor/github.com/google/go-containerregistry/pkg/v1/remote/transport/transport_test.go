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

package transport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

var (
	testReference, _ = name.NewTag("localhost:8080/user/image:latest", name.StrictValidation)
)

func TestTransportSelectionAnonymous(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	basic := &authn.Basic{Username: "foo", Password: "bar"}

	tp, err := New(testReference.Context().Registry, basic, tprt, []string{testReference.Scope(PullScope)})
	if err != nil {
		t.Errorf("New() = %v", err)
	}
	// We should get back an unmodified transport
	if tp != tprt {
		t.Errorf("New(); got %v, want %v", tp, tprt)
	}
}

func TestTransportSelectionBasic(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Basic`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	basic := &authn.Basic{Username: "foo", Password: "bar"}

	tp, err := New(testReference.Context().Registry, basic, tprt, []string{testReference.Scope(PullScope)})
	if err != nil {
		t.Errorf("New() = %v", err)
	}
	if _, ok := tp.(*basicTransport); !ok {
		t.Errorf("New(); got %T, want *basicTransport", tp)
	}
}

func TestTransportSelectionBearer(t *testing.T) {
	request := 0
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			request = request + 1
			switch request {
			case 1:
				w.Header().Set("WWW-Authenticate", `Bearer realm="http://foo.io"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			case 2:
				hdr := r.Header.Get("Authorization")
				if !strings.HasPrefix(hdr, "Basic ") {
					t.Errorf("Header.Get(Authorization); got %v, want Basic prefix", hdr)
				}
				if got, want := r.FormValue("scope"), testReference.Scope(string(PullScope)); got != want {
					t.Errorf("FormValue(scope); got %v, want %v", got, want)
				}
				// Check that we get the default value (we didn't specify it above)
				if got, want := r.FormValue("service"), testReference.RegistryStr(); got != want {
					t.Errorf("FormValue(service); got %v, want %v", got, want)
				}
				w.Write([]byte(`{"token": "dfskdjhfkhsjdhfkjhsdf"}`))
			}
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	basic := &authn.Basic{Username: "foo", Password: "bar"}
	tp, err := New(testReference.Context().Registry, basic, tprt, []string{testReference.Scope(PullScope)})
	if err != nil {
		t.Errorf("New() = %v", err)
	}
	if _, ok := tp.(*bearerTransport); !ok {
		t.Errorf("New(); got %T, want *bearerTransport", tp)
	}
}

func TestTransportSelectionBearerMissingRealm(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Bearer service="gcr.io"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	basic := &authn.Basic{Username: "foo", Password: "bar"}
	tp, err := New(testReference.Context().Registry, basic, tprt, []string{testReference.Scope(PullScope)})
	if err == nil || !strings.Contains(err.Error(), "missing realm") {
		t.Errorf("New() = %v, %v", tp, err)
	}
}

func TestTransportSelectionBearerAuthError(t *testing.T) {
	request := 0
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			request = request + 1
			switch request {
			case 1:
				w.Header().Set("WWW-Authenticate", `Bearer realm="http://foo.io"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			case 2:
				http.Error(w, "Oops", http.StatusInternalServerError)
			}
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	basic := &authn.Basic{Username: "foo", Password: "bar"}
	tp, err := New(testReference.Context().Registry, basic, tprt, []string{testReference.Scope(PullScope)})
	if err == nil {
		t.Errorf("New() = %v", tp)
	}
}

func TestTransportSelectionUnrecognizedChallenge(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Unrecognized`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	basic := &authn.Basic{Username: "foo", Password: "bar"}
	tp, err := New(testReference.Context().Registry, basic, tprt, []string{testReference.Scope(PullScope)})
	if err == nil || !strings.Contains(err.Error(), "challenge") {
		t.Errorf("New() = %v, %v", tp, err)
	}
}
