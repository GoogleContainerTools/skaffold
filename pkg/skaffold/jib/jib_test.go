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
	"path/filepath"
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

	var tests = []struct {
		name          string
		stdout        string
		expectedError bool
		expectedDeps  []string
	}{
		{
			name:          "missing version",
			stdout:        "BEGIN JIB JSON\n{\"build\":[],\"inputs\":[],\"ignore\":[]}",
			expectedError: true,
			expectedDeps:  nil,
		},
		{
			name:          "out of date version",
			stdout:        "BEGIN JIB JSON\n{\"version\":\"1.0.0\",\"build\":[],\"inputs\":[],\"ignore\":[]}",
			expectedError: true,
			expectedDeps:  nil,
		},
		{
			name:          "base case",
			stdout:        "BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[],\"inputs\":[],\"ignore\":[]}",
			expectedError: false,
			expectedDeps:  nil,
		},
		{
			stdout:        fmt.Sprintf("BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[\"%s\"],\"inputs\":[\"%s\"],\"ignore\":[]}\n", dep1, dep2),
			expectedError: false,
			expectedDeps:  []string{"dep1", "dep2"},
		},
		{
			stdout:        fmt.Sprintf("BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[],\"inputs\":[\"%s\"],\"ignore\":[]}\n", dep3),
			expectedError: false,
			expectedDeps:  []string{filepath.FromSlash("dep3/fileA"), filepath.FromSlash("dep3/sub/path/fileB")},
		},
		{
			stdout:        fmt.Sprintf("BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[],\"inputs\":[\"%s\",\"%s\",\"%s\"],\"ignore\":[]}\n", dep1, dep2, dep3),
			expectedError: false,
			expectedDeps:  []string{"dep1", "dep2", filepath.FromSlash("dep3/fileA"), filepath.FromSlash("dep3/sub/path/fileB")},
		},
		{
			stdout:        fmt.Sprintf("BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[],\"inputs\":[\"%s\",\"%s\",\"nonexistent\",\"%s\"],\"ignore\":[]}\n", dep1, dep2, dep3),
			expectedError: false,
			expectedDeps:  []string{"dep1", "dep2", filepath.FromSlash("dep3/fileA"), filepath.FromSlash("dep3/sub/path/fileB")},
		},
		{
			stdout:        fmt.Sprintf("BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[],\"inputs\":[\"%s\",\"%s\"],\"ignore\":[\"%s\"]}\n", dep1, dep2, dep2),
			expectedError: false,
			expectedDeps:  []string{"dep1"},
		},
		{
			stdout:        fmt.Sprintf("BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[\"%s\"],\"inputs\":[\"%s\"],\"ignore\":[\"%s\",\"%s\"]}\n", dep1, dep3, dep1, dep3),
			expectedError: false,
			expectedDeps:  nil,
		},
		{
			stdout:        fmt.Sprintf("BEGIN JIB JSON\n{\"version\":\"1.3.0\",\"build\":[\"%s\",\"%s\",\"%s\"],\"inputs\":[],\"ignore\":[\"%s\"]}\n", dep1, dep2, dep3, tmpDir.Path("dep3/sub/path")),
			expectedError: false,
			expectedDeps:  []string{"dep1", "dep2", filepath.FromSlash("dep3/fileA")},
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

			results, err := getDependencies(tmpDir.Root(), &exec.Cmd{Args: []string{"ignored"}, Dir: tmpDir.Root()}, "test")

			testutil.CheckErrorAndDeepEqual(t, test.expectedError, err, test.expectedDeps, results)
		})
	}
}

func TestCheckJibVersion(t *testing.T) {
	var tests = []struct {
		version       string
		expectedError bool
	}{
		{"", true},
		{"-1", true},
		{"1", true},
		{"-SNAPSHOT", true},
		{"abc", true},
		{"1.2", true},
		{"1.2-SNAPSHOT", true},
		{"1.2.0", true},
		{"1.2.0-SNAPSHOT", true},
		{"1.3.0", false},
		{"1.3.0-SNAPSHOT", false},
		{"2.0.0", false},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			err := checkJibVersion(test.version)
			testutil.CheckError(t, test.expectedError, err)
		})
	}
}
