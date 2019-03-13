/*
Copyright 2019 The Skaffold Authors

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

func TestCreateTar(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	files := map[string]string{
		"foo":     "baz1",
		"bar/bat": "baz2",
		"bar/baz": "baz3",
	}
	paths := map[string][]string{}
	for path, content := range files {
		tmpDir.Write(path, content)
		paths[path] = []string{""}
	}

	reset := testutil.Chdir(t, tmpDir.Root())
	defer reset()

	var b bytes.Buffer
	err := CreateTar(&b, ".", paths)
	testutil.CheckError(t, false, err)

	// Make sure the contents match.
	tarFiles := make(map[string]string)
	tr := tar.NewReader(&b)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		testutil.CheckError(t, false, err)

		content, err := ioutil.ReadAll(tr)
		testutil.CheckError(t, false, err)

		tarFiles[hdr.Name] = string(content)
	}

	testutil.CheckErrorAndDeepEqual(t, false, err, files, tarFiles)
}

func TestCreateTarSubDirectory(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	files := map[string]string{
		"sub/foo":     "baz1",
		"sub/bar/bat": "baz2",
		"sub/bar/baz": "baz3",
	}
	paths := map[string][]string{}
	for path, content := range files {
		tmpDir.Write(path, content)
		paths[path] = []string{""}
	}

	reset := testutil.Chdir(t, tmpDir.Root())
	defer reset()

	var b bytes.Buffer
	err := CreateTar(&b, "sub", paths)
	testutil.CheckError(t, false, err)

	// Make sure the contents match.
	tarFiles := make(map[string]string)
	tr := tar.NewReader(&b)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		testutil.CheckError(t, false, err)

		content, err := ioutil.ReadAll(tr)
		testutil.CheckError(t, false, err)

		tarFiles["sub/"+hdr.Name] = string(content)
	}

	testutil.CheckErrorAndDeepEqual(t, false, err, files, tarFiles)
}

func TestCreateTarWithAbsolutePaths(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	files := map[string]string{
		"foo":     "baz1",
		"bar/bat": "baz2",
		"bar/baz": "baz3",
	}
	paths := map[string][]string{}
	for path, content := range files {
		tmpDir.Write(path, content)
		paths[path] = []string{""}
	}

	reset := testutil.Chdir(t, tmpDir.Root())
	defer reset()

	var b bytes.Buffer
	err := CreateTar(&b, tmpDir.Root(), paths)
	testutil.CheckError(t, false, err)

	// Make sure the contents match.
	tarFiles := make(map[string]string)
	tr := tar.NewReader(&b)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		testutil.CheckError(t, false, err)

		content, err := ioutil.ReadAll(tr)
		testutil.CheckError(t, false, err)

		tarFiles[hdr.Name] = string(content)
	}

	testutil.CheckErrorAndDeepEqual(t, false, err, files, tarFiles)
}

func TestCreateTarWithExplicitDestination(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	contentByName := map[string]string{
		"foo":     "baz1",
		"bar/bat": "baz2",
		"bar/baz": "baz3",
	}
	paths := map[string][]string{
		"foo":     {"f1", "d1/f1"},
		"bar/bat": {"d2/f2", "d3/f2"},
		"bar/baz": {"d2/f3", "f2", "d1/f2"},
	}
	for path, content := range contentByName {
		tmpDir.Write(path, content)
	}
	expected := map[string]string{
		"f1":    "baz1",
		"d1/f1": "baz1",
		"d2/f2": "baz2",
		"d3/f2": "baz2",
		"d2/f3": "baz3",
		"d1/f2": "baz3",
		"f2":    "baz3",
	}

	reset := testutil.Chdir(t, tmpDir.Root())
	defer reset()

	var b bytes.Buffer
	err := CreateTar(&b, ".", paths)
	testutil.CheckError(t, false, err)

	// Make sure the contents match.
	tarFiles := make(map[string]string)
	tr := tar.NewReader(&b)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		testutil.CheckError(t, false, err)

		content, err := ioutil.ReadAll(tr)
		testutil.CheckError(t, false, err)

		tarFiles[hdr.Name] = string(content)
	}

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, tarFiles)
}
