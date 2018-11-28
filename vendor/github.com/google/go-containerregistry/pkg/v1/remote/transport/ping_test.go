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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/google/go-containerregistry/pkg/name"
)

var (
	testRegistry, _ = name.NewRegistry("localhost:8080", name.StrictValidation)
)

func TestChallengeParsing(t *testing.T) {
	tests := []struct {
		input  string
		output map[string]string
	}{{
		input: `foo="bar"`,
		output: map[string]string{
			"foo": "bar",
		},
	}, {
		input: `foo`,
		output: map[string]string{
			"foo": "",
		},
	}, {
		input: `foo="bar",baz="blah"`,
		output: map[string]string{
			"foo": "bar",
			"baz": "blah",
		},
	}, {
		input: `baz="blah", foo="bar"`,
		output: map[string]string{
			"foo": "bar",
			"baz": "blah",
		},
	}, {
		input: `realm="https://gcr.io/v2/token", service="gcr.io", scope="repository:foo/bar:pull"`,
		output: map[string]string{
			"realm":   "https://gcr.io/v2/token",
			"service": "gcr.io",
			"scope":   "repository:foo/bar:pull",
		},
	}}

	for _, test := range tests {
		params := parseChallenge(test.input)
		if diff := cmp.Diff(test.output, params); diff != "" {
			t.Errorf("parseChallenge(%s); (-want +got) %s", test.input, diff)
		}
	}
}

func TestPingNoChallenge(t *testing.T) {
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

	pr, err := ping(testRegistry, tprt)
	if err != nil {
		t.Errorf("ping() = %v", err)
	}
	if pr.challenge != anonymous {
		t.Errorf("ping(); got %v, want %v", pr.challenge, anonymous)
	}
}

func TestPingBasicChallengeNoParams(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `BASIC`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	pr, err := ping(testRegistry, tprt)
	if err != nil {
		t.Errorf("ping() = %v", err)
	}
	if pr.challenge != basic {
		t.Errorf("ping(); got %v, want %v", pr.challenge, basic)
	}
	if got, want := len(pr.parameters), 0; got != want {
		t.Errorf("ping(); got %v, want %v", got, want)
	}
}

func TestPingBearerChallengeWithParams(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="http://auth.foo.io/token`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	pr, err := ping(testRegistry, tprt)
	if err != nil {
		t.Errorf("ping() = %v", err)
	}
	if pr.challenge != bearer {
		t.Errorf("ping(); got %v, want %v", pr.challenge, bearer)
	}
	if got, want := len(pr.parameters), 1; got != want {
		t.Errorf("ping(); got %v, want %v", got, want)
	}
}

func TestUnsupportedStatus(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="http://auth.foo.io/token`)
			http.Error(w, "Forbidden", http.StatusForbidden)
		}))
	defer server.Close()
	tprt := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	pr, err := ping(testRegistry, tprt)
	if err == nil {
		t.Errorf("ping() = %v", pr)
	}
}
