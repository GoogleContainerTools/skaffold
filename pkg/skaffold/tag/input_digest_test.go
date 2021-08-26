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

func TestInputDigest_GenerateCorrectChecksumForSingleFile(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		dir := t.TempDir()
		cwdBackup, err := os.Getwd()
		t.RequireNoError(err)
		t.RequireNoError(os.Chdir(dir))
		defer func() { t.RequireNoError(os.Chdir(cwdBackup)) }()

		filePath := "temp.file"
		t.RequireNoError(ioutil.WriteFile(filePath, []byte("hello\ngo\n"), 0644))

		expectedDigest := "3cced2dec96a8b41b22875686d8941a9"
		relPathHash, err := fileHasher(filePath, ".")
		t.CheckErrorAndDeepEqual(false, err, expectedDigest, relPathHash)
		absPathHash, err := fileHasher(filepath.Join(dir, filePath), dir)
		t.CheckErrorAndDeepEqual(false, err, expectedDigest, absPathHash)
	})
}
