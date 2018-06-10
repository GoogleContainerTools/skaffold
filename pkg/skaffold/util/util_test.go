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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSupportedKubernetesFormats(t *testing.T) {
	var tests = []struct {
		description string
		in          string
		out         bool
	}{
		{
			description: "yaml",
			in:          "filename.yaml",
			out:         true,
		},
		{
			description: "yml",
			in:          "filename.yml",
			out:         true,
		},
		{
			description: "json",
			in:          "filename.json",
			out:         true,
		},
		{
			description: "txt",
			in:          "filename.txt",
			out:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actual := IsSupportedKubernetesFormat(tt.in)
			if tt.out != actual {
				t.Errorf("out: %t, actual: %t", tt.out, actual)
			}
		})
	}
}

func TestExpandPathsGlob(t *testing.T) {
	tmp, cleanup := testutil.TempDir(t)
	defer cleanup()

	os.MkdirAll(filepath.Join(tmp, "dir", "sub_dir"), 0700)
	os.MkdirAll(filepath.Join(tmp, "dir_b", "sub_dir_b"), 0700)
	ioutil.WriteFile(filepath.Join(tmp, "dir_b", "sub_dir_b", "file"), []byte(""), 0650)
	ioutil.WriteFile(filepath.Join(tmp, "dir", "sub_dir", "file"), []byte(""), 0650)

	var tests = []struct {
		description string
		in          []string
		out         []string
		shouldErr   bool
	}{
		{
			description: "match exact filename",
			in:          []string{"dir/sub_dir/file"},
			out:         []string{filepath.Join(tmp, "dir", "sub_dir", "file")},
		},
		{
			description: "match leaf directory glob",
			in:          []string{"dir/sub_dir/*"},
			out:         []string{filepath.Join(tmp, "dir", "sub_dir", "file")},
		},
		{
			description: "match top level glob",
			in:          []string{"dir*"},
			out:         []string{filepath.Join(tmp, "dir", "sub_dir", "file"), filepath.Join(tmp, "dir_b", "sub_dir_b", "file")},
		},
		{
			description: "error unmatched glob",
			in:          []string{"dir/sub_dir_c/*"},
			shouldErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actual, err := ExpandPathsGlob(tmp, tt.in)

			testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.out, actual)
		})
	}
}

func TestDefaultConfigFilenameAlternate(t *testing.T) {
	testDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	files := map[string]string{
		"skaffold.yml": "foo",
	}
	if err := setupFiles(testDir, files); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}

	for file := range files {
		path := filepath.Join(testDir, "skaffold.yaml")
		expectedContents := files[file]
		actualContents, err := ReadConfiguration(path)
		if err != nil {
			t.Errorf("Error '%s' reading configuration file %s", err, path)
		}
		if expectedContents != string(actualContents) {
			t.Errorf("File contents don't match. %s != %s", actualContents, expectedContents)
		}
	}
}
