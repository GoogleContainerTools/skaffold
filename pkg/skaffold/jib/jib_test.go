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

package jib

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependencies(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("dep1", "")
	tmpDir.Write("dep2", "")
	tmpDir.Write("dep3/fileA", "")
	tmpDir.Write("dep3/sub/path/fileB", "")

	dep1 := tmpDir.Path("dep1")
	dep2 := tmpDir.Path("dep2")
	dep3 := tmpDir.Path("dep3")
	dep3FileA := tmpDir.Path("dep3/fileA")
	dep3Sub := tmpDir.Path("dep3/sub")
	dep3SubPath := tmpDir.Path("dep3/sub/path")
	dep3SubPathFileB := tmpDir.Path("dep3/sub/path/fileB")

	var tests = []struct {
		stdout       string
		expectedDeps []string
	}{
		{
			stdout:       "",
			expectedDeps: nil,
		},
		{
			stdout:       fmt.Sprintf("%s\n%s", dep1, dep2),
			expectedDeps: []string{dep1, dep2},
		},
		{
			stdout:       fmt.Sprintf("%s\n%s\n", dep1, dep2),
			expectedDeps: []string{dep1, dep2},
		},
		{
			stdout:       fmt.Sprintf("%s\n%s\n%s\n", dep1, dep2, tmpDir.Root()),
			expectedDeps: []string{dep1, dep2},
		},
		{
			stdout:       "\n\n\n",
			expectedDeps: nil,
		},
		{
			stdout:       fmt.Sprintf("\n\n%s\n\n%s\n\n\n", dep1, dep2),
			expectedDeps: []string{dep1, dep2},
		},
		{
			stdout:       dep3,
			expectedDeps: []string{dep3, dep3FileA, dep3Sub, dep3SubPath, dep3SubPathFileB},
		},
		{
			stdout:       fmt.Sprintf("%s\n%s\n%s\n", dep1, dep2, dep3),
			expectedDeps: []string{dep1, dep2, dep3, dep3FileA, dep3Sub, dep3SubPath, dep3SubPathFileB},
		},
		{
			stdout:       fmt.Sprintf("%s\nnonexistent\n%s\n%s\n", dep1, dep2, dep3),
			expectedDeps: []string{dep1, dep2, dep3, dep3FileA, dep3Sub, dep3SubPath, dep3SubPathFileB},
		},
	}

	for _, test := range tests {
		t.Run("getDependencies", func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOut(
				"ignored",
				test.stdout,
			)

			deps, err := getDependencies(&exec.Cmd{Args: []string{"ignored"}, Dir: tmpDir.Root()})

			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedDeps, deps)
		})
	}
}
