/*
Copyright 2018 The Skaffold Authors

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

package util

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func Test_addFileToTar(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	files := map[string]string{
		"foo":     "baz1",
		"bar/bat": "baz2",
		"bar/baz": "baz3",
	}
	for p, c := range files {
		tmpDir.Write(p, c)
	}

	// Add all the files to a tar.
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	for p := range files {
		path := tmpDir.Path(p)
		if err := addFileToTar(path, p, tw); err != nil {
			t.Fatalf("addFileToTar() error = %v", err)
		}
	}
	tw.Close()

	// Make sure the contents match.
	tr := tar.NewReader(&b)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Error reading tar: %s", err)
		}
		expectedContents, ok := files[hdr.Name]
		if !ok {
			t.Errorf("Unexpected file in tar: %s", hdr.Name)
		}
		actualContents, err := ioutil.ReadAll(tr)
		if err != nil {
			t.Errorf("Error %s reading file %s from tar", err, hdr.Name)
		}
		if expectedContents != string(actualContents) {
			t.Errorf("File contents don't match. %s != %s", actualContents, expectedContents)
		}
	}
}
