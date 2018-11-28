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

	"github.com/google/go-cmp/cmp"
)

func TestGoodManifestSimple(t *testing.T) {
	got, err := ParseManifest(strings.NewReader(`{}`))
	if err != nil {
		t.Errorf("Unexpected error parsing manifest: %v", err)
	}

	want := Manifest{}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("ParseManifest({}); (-want +got) %s", diff)
	}
}

func TestGoodManifestWithHash(t *testing.T) {
	good, err := ParseManifest(strings.NewReader(`{
  "config": {
    "digest": "sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
  }
}`))
	if err != nil {
		t.Errorf("Unexpected error parsing manifest: %v", err)
	}

	if got, want := good.Config.Digest.Algorithm, "sha256"; got != want {
		t.Errorf("ParseManifest().Config.Digest.Algorithm; got %v, want %v", got, want)
	}
}

func TestManifestWithBadHash(t *testing.T) {
	bad, err := ParseManifest(strings.NewReader(`{
  "config": {
    "digest": "sha256:deadbeed"
  }
}`))
	if err == nil {
		t.Errorf("Expected error parsing manifest, but got: %v", bad)
	}
}
