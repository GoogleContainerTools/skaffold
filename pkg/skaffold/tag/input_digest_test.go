/*
Copyright 2020 The Skaffold Authors

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

package tag

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInputDigest(t *testing.T) {
	fileContents1, fileContents2 := []byte("hello\ngo\n"), []byte("bye\ngo\n")

	testutil.Run(t, "SameDigestForRelAndAbsPath", func(t *testutil.T) {
		dir := t.TempDir()
		cwdBackup, err := os.Getwd()
		t.RequireNoError(err)
		t.RequireNoError(os.Chdir(dir))
		defer func() { t.RequireNoError(os.Chdir(cwdBackup)) }()

		file := "temp.file"
		t.RequireNoError(ioutil.WriteFile(file, fileContents1, 0644))

		relPathHash, err := fileHasher(file, ".")
		t.CheckErrorAndDeepEqual(false, err, "3cced2dec96a8b41b22875686d8941a9", relPathHash)
		absPathHash, err := fileHasher(filepath.Join(dir, file), dir)
		t.CheckErrorAndDeepEqual(false, err, relPathHash, absPathHash)
	})

	testutil.Run(t, "SameDigestForTwoDifferentAbsPaths", func(t *testutil.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		file1, file2 := filepath.Join(dir1, "temp.file"), filepath.Join(dir2, "temp.file")
		t.RequireNoError(ioutil.WriteFile(file1, fileContents1, 0644))
		t.RequireNoError(ioutil.WriteFile(file2, fileContents1, 0644))

		hash1, err := fileHasher(file1, dir1)
		t.CheckErrorAndDeepEqual(false, err, "3cced2dec96a8b41b22875686d8941a9", hash1)
		hash2, err := fileHasher(file2, dir2)
		t.CheckErrorAndDeepEqual(false, err, hash1, hash2)
	})

	testutil.Run(t, "DifferentDigestForDifferentFilenames", func(t *testutil.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		file1, file2 := filepath.Join(dir1, "temp1.file"), filepath.Join(dir2, "temp2.file")
		t.RequireNoError(ioutil.WriteFile(file1, fileContents1, 0644))
		t.RequireNoError(ioutil.WriteFile(file2, fileContents1, 0644))

		hash1, err := fileHasher(file1, dir1)
		t.CheckNoError(err)
		hash2, err := fileHasher(file2, dir2)
		t.CheckNoError(err)
		t.CheckFalse(hash1 == hash2)
	})

	testutil.Run(t, "DifferentDigestForDifferentContent", func(t *testutil.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		file1, file2 := filepath.Join(dir1, "temp.file"), filepath.Join(dir2, "temp.file")
		t.RequireNoError(ioutil.WriteFile(file1, fileContents1, 0644))
		t.RequireNoError(ioutil.WriteFile(file2, fileContents2, 0644))

		hash1, err := fileHasher(file1, dir1)
		t.CheckNoError(err)
		hash2, err := fileHasher(file2, dir2)
		t.CheckNoError(err)
		t.CheckFalse(hash1 == hash2)
	})
}
