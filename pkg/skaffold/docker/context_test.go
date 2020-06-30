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

package docker

import (
	"archive/tar"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildContext(t *testing.T) {
	for _, dir := range []string{".", "sub"} {
		testutil.Run(t, dir, func(t *testutil.T) {
			t.NewTempDir().
				Write(dir+"/.dockerignore", "**/ignored.txt\nalsoignored.txt").
				Touch(dir + "/Dockerfile").
				Touch(dir + "/files/ignored.txt").
				Touch(dir + "/files/included.txt").
				Touch(dir + "/ignored.txt").
				Touch(dir + "/alsoignored.txt").
				Chdir()

			buildCtx, relDockerfile, err := BuildContext(dir, "Dockerfile")
			t.CheckNoError(err)
			t.CheckDeepEqual("Dockerfile", relDockerfile)

			files, err := readFiles(buildCtx)
			t.CheckNoError(err)
			t.CheckFalse(files["ignored.txt"])
			t.CheckFalse(files["alsoignored.txt"])
			t.CheckFalse(files["files/ignored.txt"])
			t.CheckTrue(files[".dockerignore"])
			t.CheckTrue(files["files/included.txt"])
			t.CheckTrue(files["Dockerfile"])
		})
	}
}

func TestBuildContextDockerfileOutsideContext(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().
			Touch("Dockerfile").
			Touch("sub/file.txt").
			Chdir()

		buildCtx, relDockerfile, err := BuildContext("sub", "../Dockerfile")
		t.CheckNoError(err)

		files, err := readFiles(buildCtx)
		t.CheckNoError(err)
		t.CheckTrue(files["file.txt"])
		t.CheckTrue(files[".dockerignore"]) // Created on the fly by BuildContext()
		t.CheckTrue(files[relDockerfile])
	})
}

func TestBuildContextDockerfileNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Chdir()

		_, _, err := BuildContext(".", "Dockerfile.notfound")

		t.CheckError(true, err)
	})
}

func readFiles(buildCtx io.ReadCloser) (map[string]bool, error) {
	defer buildCtx.Close()

	files := make(map[string]bool)

	tr := tar.NewReader(buildCtx)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		files[header.Name] = true
	}

	return files, nil
}
