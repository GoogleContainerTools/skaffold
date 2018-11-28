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
	"testing"
)

func TestReader(t *testing.T) {
	want := "This is the input string."
	buf := bytes.NewBufferString(want)
	zipped, err := GzipReadCloser(NopReadCloser(buf))
	if err != nil {
		t.Errorf("GzipReadCloser() = %v", err)
	}
	unzipped, err := GunzipReadCloser(zipped)
	if err != nil {
		t.Errorf("GunzipReadCloser() = %v", err)
	}

	b, err := ioutil.ReadAll(unzipped)
	if err != nil {
		t.Errorf("ReadAll() = %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("ReadAll(); got %q, want %q", got, want)
	}
}
