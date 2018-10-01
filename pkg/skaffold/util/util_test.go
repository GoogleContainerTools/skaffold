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
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("dir/sub_dir/file", "")
	tmpDir.Write("dir_b/sub_dir_b/file", "")

	var tests = []struct {
		description string
		in          []string
		out         []string
		shouldErr   bool
	}{
		{
			description: "match exact filename",
			in:          []string{"dir/sub_dir/file"},
			out:         []string{tmpDir.Path("dir/sub_dir/file")},
		},
		{
			description: "match leaf directory glob",
			in:          []string{"dir/sub_dir/*"},
			out:         []string{tmpDir.Path("dir/sub_dir/file")},
		},
		{
			description: "match top level glob",
			in:          []string{"dir*"},
			out:         []string{tmpDir.Path("dir/sub_dir/file"), tmpDir.Path("dir_b/sub_dir_b/file")},
		},
		{
			description: "error unmatched glob",
			in:          []string{"dir/sub_dir_c/*"},
			shouldErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actual, err := ExpandPathsGlob(tmpDir.Root(), tt.in)

			testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.out, actual)
		})
	}
}

func TestDefaultConfigFilenameAlternate(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("skaffold.yml", "foo")

	content, err := ReadConfiguration(tmpDir.Path("skaffold.yaml"))

	testutil.CheckErrorAndDeepEqual(t, false, err, []byte("foo"), content)
}

func TestExpand(t *testing.T) {
	var tests = []struct {
		description string
		text        string
		key         string
		value       string
		expected    string
	}{
		{
			description: "${key} syntax",
			text:        "BEFORE[${key}]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[VALUE]AFTER",
		},
		{
			description: "$key syntax",
			text:        "BEFORE[$key]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[VALUE]AFTER",
		},
		{
			description: "replace all",
			text:        "BEFORE[$key][${key}][$key][${key}]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[VALUE][VALUE][VALUE][VALUE]AFTER",
		},
		{
			description: "ignore common prefix",
			text:        "BEFORE[$key1][${key1}]AFTER",
			key:         "key",
			value:       "VALUE",
			expected:    "BEFORE[$key1][${key1}]AFTER",
		},
		{
			description: "just the ${key} placeholder",
			text:        "${key}",
			key:         "key",
			value:       "VALUE",
			expected:    "VALUE",
		},
		{
			description: "just the $key placeholder",
			text:        "$key",
			key:         "key",
			value:       "VALUE",
			expected:    "VALUE",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := Expand(test.text, test.key, test.value)

			testutil.CheckDeepEqual(t, test.expected, actual)
		})
	}
}
