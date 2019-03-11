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
			stdout:       "BEGIN JIB JSON\n{\"build\":[],\"inputs\":[],\"ignore\":[]}",
			expectedDeps: nil,
		},
		{
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\"],\"inputs\":[\"%s\"],\"ignore\":[]}\n", dep1, dep2),
			expectedDeps: []string{dep1, dep2},
		},
		{
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\"],\"ignore\":[]}\n", dep3),
			expectedDeps: []string{dep3, dep3FileA, dep3Sub, dep3SubPath, dep3SubPathFileB},
		},
		{
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\",\"%s\",\"%s\"],\"ignore\":[]}\n", dep1, dep2, dep3),
			expectedDeps: []string{dep1, dep2, dep3, dep3FileA, dep3Sub, dep3SubPath, dep3SubPathFileB},
		},
		{
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\",\"%s\",\"nonexistent\",\"%s\"],\"ignore\":[]}\n", dep1, dep2, dep3),
			expectedDeps: []string{dep1, dep2, dep3, dep3FileA, dep3Sub, dep3SubPath, dep3SubPathFileB},
		},
		{
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\",\"%s\"],\"ignore\":[\"%s\"]}\n", dep1, dep2, dep2),
			expectedDeps: []string{dep1},
		},
		{
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\"],\"inputs\":[\"%s\"],\"ignore\":[\"%s\",\"%s\"]}\n", dep1, dep3, dep1, dep3),
			expectedDeps: nil,
		},
		{
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\",\"%s\",\"%s\"],\"inputs\":[],\"ignore\":[\"%s\"]}\n", dep1, dep2, dep3, dep3SubPath),
			expectedDeps: []string{dep1, dep2, dep3, dep3FileA, dep3Sub},
		},
	}

	for _, test := range tests {
		// Reset map between each test to ensure stdout is read each time
		watchedFiles = map[string]filesLists{}

		t.Run("getDependencies", func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOut(
				"ignored",
				test.stdout,
			)

			results, err := getDependencies(&exec.Cmd{Args: []string{"ignored"}, Dir: tmpDir.Root()}, "test")

			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedDeps, results)
		})
	}
}
