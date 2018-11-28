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

package v1util

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1"
)

func mustHash(s string, t *testing.T) v1.Hash {
	h, _, err := v1.SHA256(strings.NewReader(s))
	if err != nil {
		t.Fatalf("SHA256(%s) = %v", s, err)
	}
	return h
}

func TestVerificationFailure(t *testing.T) {
	want := "This is the input string."
	buf := bytes.NewBufferString(want)

	verified, err := VerifyReadCloser(NopReadCloser(buf), mustHash("not the same", t))
	if err != nil {
		t.Fatalf("VerifyReadCloser() = %v", err)
	}
	if b, err := ioutil.ReadAll(verified); err == nil {
		t.Errorf("ReadAll() = %q; want verification error", string(b))
	}
}

func TestVerification(t *testing.T) {
	want := "This is the input string."
	buf := bytes.NewBufferString(want)

	verified, err := VerifyReadCloser(NopReadCloser(buf), mustHash(want, t))
	if err != nil {
		t.Fatalf("VerifyReadCloser() = %v", err)
	}
	if _, err := ioutil.ReadAll(verified); err != nil {
		t.Errorf("ReadAll() = %v", err)
	}
}
