// +build !windows

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
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCanonical(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("skaffold.yml", "foo")
	err := os.Symlink(tmpDir.Path("skaffold.yml"), tmpDir.Path("newfile"))
	testutil.CheckError(t, false, err)

	// must canonicalize both as the tmpDir may be a symlinked directory
	var filepath = Canonical(tmpDir.Path("skaffold.yml"))
	var canonical = Canonical(tmpDir.Path("newfile"))
	testutil.CheckDeepEqual(t, filepath, canonical)
}
