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
	"compress/gzip"
	"io"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCreateTarWithParents(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		files := map[string]string{
			"foo":     "baz1",
			"bar/bat": "baz2",
			"bar/baz": "baz3",
		}
		_, paths := prepareFiles(t, ".", files)

		var b bytes.Buffer
		err := CreateTarWithParents(&b, ".", paths, 10, 100, time.Now())
		t.CheckNoError(err)

		// Make sure the contents match.
		tarFiles := make(map[string]string)
		tr := tar.NewReader(&b)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			t.CheckNoError(err)
			t.CheckDeepEqual(10, hdr.Uid)
			t.CheckDeepEqual(100, hdr.Gid)

			content, err := ioutil.ReadAll(tr)
			t.CheckNoError(err)

			tarFiles[hdr.Name] = string(content)
		}

		t.CheckDeepEqual(map[string]string{
			"foo":     "baz1",
			"bar":     "",
			"bar/bat": "baz2",
			"bar/baz": "baz3",
		}, tarFiles)
	})
}

func TestCreateTarGz(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		files := map[string]string{
			"foo":     "baz1",
			"bar/bat": "baz2",
			"bar/baz": "baz3",
		}
		_, paths := prepareFiles(t, ".", files)

		var b bytes.Buffer
		err := CreateTarGz(&b, ".", paths)
		t.CheckNoError(err)

		// Make sure the contents match.
		tarFiles, err := untarFiles(&b)
		t.CheckNoError(err)
		t.CheckDeepEqual(files, tarFiles)
	})
}

func TestCreateTarGzSubDirectory(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		files := map[string]string{
			"foo":     "baz1",
			"bar/bat": "baz2",
			"bar/baz": "baz3",
		}
		_, paths := prepareFiles(t, "sub", files)

		var b bytes.Buffer
		err := CreateTarGz(&b, "sub", paths)
		t.CheckNoError(err)

		// Make sure the contents match.
		tarFiles, err := untarFiles(&b)
		t.CheckNoError(err)
		t.CheckDeepEqual(files, tarFiles)
	})
}

func TestCreateTarGzEmptyFolder(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().
			Mkdir("empty").
			Chdir()

		var b bytes.Buffer
		err := CreateTarGz(&b, ".", []string{"empty"})
		t.CheckNoError(err)

		// Make sure the contents match.
		tarFiles, err := untarFiles(&b)
		t.CheckNoError(err)
		t.CheckDeepEqual(map[string]string{"empty": ""}, tarFiles)
	})
}

func TestCreateTarGzWithAbsolutePaths(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		files := map[string]string{
			"foo":     "baz1",
			"bar/bat": "baz2",
			"bar/baz": "baz3",
		}
		tmpDir, paths := prepareFiles(t, ".", files)

		var b bytes.Buffer
		err := CreateTarGz(&b, tmpDir.Root(), tmpDir.Paths(paths...))
		t.CheckNoError(err)

		// Make sure the contents match.
		tarFiles, err := untarFiles(&b)
		t.CheckNoError(err)
		t.CheckDeepEqual(files, tarFiles)
	})
}

func TestAddFileToTarSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires extra privileges on Windows")
	}

	testutil.Run(t, "", func(t *testutil.T) {
		files := map[string]string{
			"foo":     "baz1",
			"bar/bat": "baz2",
			"bar/baz": "baz3",
		}
		tmpDir, paths := prepareFiles(t, ".", files)

		links := map[string]string{
			"foo.link":     "foo",
			"bat.link":     "bar/bat",
			"bat/baz.link": "bar/baz",
		}
		for link, file := range links {
			tmpDir.Symlink(file, link)
			paths = append(paths, link)
		}

		var b bytes.Buffer
		err := CreateTarGz(&b, tmpDir.Root(), tmpDir.Paths(paths...))
		t.CheckNoError(err)

		// Make sure the links match.
		gzr, err := gzip.NewReader(&b)
		t.CheckNoError(err)
		tr := tar.NewReader(gzr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			t.CheckNoError(err)

			if _, isFile := files[hdr.Name]; isFile {
				continue
			}

			link, isLink := links[hdr.Name]
			if !isLink {
				t.Errorf("Unexpected file/link in tar: %s", hdr.Name)
			}
			if hdr.Linkname != link {
				t.Errorf("Link destination doesn't match. %s != %s.", link, hdr.Linkname)
			}
		}
	})
}

func prepareFiles(t *testutil.T, folder string, files map[string]string) (*testutil.TempDir, []string) {
	tmpDir := t.NewTempDir().Chdir()

	var paths []string
	for file, content := range files {
		path := filepath.Join(folder, file)
		tmpDir.Write(path, content)
		paths = append(paths, path)
	}

	return tmpDir, paths
}

func untarFiles(r io.Reader) (map[string]string, error) {
	files := make(map[string]string)

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		content, err := ioutil.ReadAll(tr)
		if err != nil {
			return nil, err
		}

		files[hdr.Name] = string(content)
	}

	return files, nil
}
