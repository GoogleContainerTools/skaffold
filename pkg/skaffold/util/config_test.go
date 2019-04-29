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
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestReadConfiguration(t *testing.T) {
	localFile, delete := testutil.TempFile(t, "skaffold.yaml", []byte("some yaml"))
	defer delete()

	content, err := ReadConfiguration(localFile)

	testutil.CheckErrorAndDeepEqual(t, false, err, []byte("some yaml"), content)
}

func TestReadConfigurationFallback(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	reset := testutil.Chdir(t, tmpDir.Root())
	defer reset()

	// skaffold.yaml doesn't exist but .yml does
	tmpDir.Write("skaffold.yml", "some yaml")

	content, err := ReadConfiguration("skaffold.yaml")

	testutil.CheckErrorAndDeepEqual(t, false, err, []byte("some yaml"), content)
}

func TestReadConfigurationNotFound(t *testing.T) {
	_, err := ReadConfiguration("unknown-config-file.yaml")

	testutil.CheckError(t, true, err)
	if !os.IsNotExist(err) {
		t.Error("error should say that file doesn't exist")
	}
}

func TestReadConfigurationRemote(t *testing.T) {
	remoteFile, teardown := testutil.ServeFile(t, []byte("remote file"))
	defer teardown()

	content, err := ReadConfiguration(remoteFile)

	testutil.CheckErrorAndDeepEqual(t, false, err, []byte("remote file"), content)
}
