/*
Copyright 2021 The Skaffold Authors

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

package list

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFiles(t *testing.T) {
	tmpDir := testutil.NewTempDir(t).
		Touch(
			"bar.yaml",
			"baz.yaml",
			"foo.go",
			"sub1/bar.yaml",
			"sub1/baz.yaml",
			"sub1/foo.go",
			"sub1/sub2/bar.yaml",
			"sub1/sub2/baz.yaml",
			"sub1/sub2/foo.go",
		)
	tests := []struct {
		description   string
		workspace     string
		patterns      []string
		excludes      []string
		shouldErr     bool
		expectedFiles []string
	}{
		{
			description: "watch nothing",
			workspace:   tmpDir.Root(),
		},
		{
			description: "error on no matches",
			patterns:    []string{"this-pattern-does-not-match-any-files"},
			shouldErr:   true,
		},
		{
			description: "include all files using path",
			workspace:   tmpDir.Root(),
			patterns:    []string{"."},
			excludes:    nil,
			expectedFiles: []string{
				"bar.yaml",
				"baz.yaml",
				"foo.go",
				"sub1/bar.yaml",
				"sub1/baz.yaml",
				"sub1/foo.go",
				"sub1/sub2/bar.yaml",
				"sub1/sub2/baz.yaml",
				"sub1/sub2/foo.go",
			},
		},
		{
			description: "include all files using non-globstar wildcard",
			// this test case seems odd, but it is current behavior
			workspace: tmpDir.Root(),
			patterns:  []string{"*"},
			excludes:  nil,
			expectedFiles: []string{
				"bar.yaml",
				"baz.yaml",
				"foo.go",
				"sub1/bar.yaml",
				"sub1/baz.yaml",
				"sub1/foo.go",
				"sub1/sub2/bar.yaml",
				"sub1/sub2/baz.yaml",
				"sub1/sub2/foo.go",
			},
		},
		{
			description: "include all files using globstar",
			workspace:   tmpDir.Root(),
			patterns:    []string{"**"},
			excludes:    nil,
			expectedFiles: []string{
				"bar.yaml",
				"baz.yaml",
				"foo.go",
				"sub1/bar.yaml",
				"sub1/baz.yaml",
				"sub1/foo.go",
				"sub1/sub2/bar.yaml",
				"sub1/sub2/baz.yaml",
				"sub1/sub2/foo.go",
			},
		},
		{
			description: "globstar pattern with file extension matching",
			workspace:   tmpDir.Root(),
			patterns:    []string{"**/*.yaml"},
			excludes:    nil,
			expectedFiles: []string{
				"bar.yaml",
				"baz.yaml",
				"sub1/bar.yaml",
				"sub1/baz.yaml",
				"sub1/sub2/bar.yaml",
				"sub1/sub2/baz.yaml",
			},
		},
		{
			description: "non-globstar wildcard pattern with file extension match does not recurse subdirectories",
			workspace:   tmpDir.Root(),
			patterns:    []string{"*/*.go"},
			excludes:    nil,
			expectedFiles: []string{
				"sub1/foo.go",
			},
		},
		{
			description: "globstar pattern recurses multiple levels of subdirectories",
			workspace:   tmpDir.Root(),
			patterns:    []string{"**/*.go"},
			excludes:    nil,
			expectedFiles: []string{
				"foo.go",
				"sub1/foo.go",
				"sub1/sub2/foo.go",
			},
		},
		{
			description: "globstar excludes recurses multiple levels of subdirectories",
			workspace:   tmpDir.Root(),
			patterns:    []string{"**/*.yaml"},
			excludes:    []string{"**/baz.yaml"},
			expectedFiles: []string{
				"bar.yaml",
				"sub1/bar.yaml",
				"sub1/sub2/bar.yaml",
			},
		},
		{
			description: "include and exclude all",
			workspace:   tmpDir.Root(),
			patterns:    []string{"**"},
			excludes:    []string{"**"},
		},
		{
			description: "include and exclude all with globstar and file extension matching",
			workspace:   tmpDir.Root(),
			patterns:    []string{"**/*.go"},
			excludes:    []string{"**/*.go"},
		},
		{
			description: "avoid duplicates for overlapping patterns",
			workspace:   tmpDir.Root(),
			patterns: []string{
				"**/*.go",
				"*.go",
				"*/*.go",
				"*/*/*.go",
				"sub1/*.go",
				"sub1/sub2/*.go",
			},
			excludes: nil,
			expectedFiles: []string{
				"foo.go",
				"sub1/foo.go",
				"sub1/sub2/foo.go",
			},
		},
		{
			description: "workspace is relative path",
			workspace:   ".",
			patterns: []string{
				".",
			},
			excludes: nil,
			expectedFiles: []string{
				"list.go",
				"list_test.go",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			files, err := Files(test.workspace, test.patterns, test.excludes)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedFiles, files,
				cmpopts.AcyclicTransformer("separator", filepath.FromSlash),
			)
		})
	}
}
