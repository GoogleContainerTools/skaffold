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

package v1

import (
	"strings"
	"testing"
)

func TestGoodHashes(t *testing.T) {
	good := []string{
		"sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	for _, s := range good {
		h, err := NewHash(s)
		if err != nil {
			t.Errorf("Unexpected error parsing hash: %v", err)
		}
		if got, want := h.String(), s; got != want {
			t.Errorf("String(); got %q, want %q", got, want)
		}
	}
}

func TestBadHashes(t *testing.T) {
	bad := []string{
		// Too short
		"sha256:deadbeef",
		// Bad character
		"sha256:o123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		// Unknown algorithm
		"md5:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		// Too few parts
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		// Too many parts
		"md5:sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	for _, s := range bad {
		h, err := NewHash(s)
		if err == nil {
			t.Errorf("Expected error, got: %v", h)
		}
	}
}

func TestSHA256(t *testing.T) {
	input := "asdf"
	h, n, err := SHA256(strings.NewReader(input))
	if err != nil {
		t.Errorf("SHA256(asdf) = %v", err)
	}
	if got, want := h.Algorithm, "sha256"; got != want {
		t.Errorf("Algorithm; got %v, want %v", got, want)
	}
	if got, want := h.Hex, "f0e4c2f76c58916ec258f246851bea091d14d4247a2fc3e18694461b1816e13b"; got != want {
		t.Errorf("Hex; got %v, want %v", got, want)
	}
	if got, want := n, int64(len(input)); got != want {
		t.Errorf("n; got %v, want %v", got, want)
	}
}
