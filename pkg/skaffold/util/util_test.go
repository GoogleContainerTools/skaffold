/*
Copyright 2018 Google LLC

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
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/testutil"
	"github.com/spf13/afero"
)

func TestExpandDeps(t *testing.T) {
	var tests = []struct {
		description string
		in          []string
		out         []string
		shouldErr   bool
	}{
		{
			description: "add single files",
			in:          []string{"test/a", "test/b", "test/c"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "add directory",
			in:          []string{"test"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "add directory trailing slash",
			in:          []string{"test/"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "file not exist",
			in:          []string{"test/d"},
			shouldErr:   true,
		},
		{
			description: "add wildcard star",
			in:          []string{"*"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "add wildcard any character",
			in:          []string{"test/?"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
	}

	defer ResetFs()
	Fs = afero.NewMemMapFs()

	Fs.MkdirAll("test", 0755)
	files := []string{"test/a", "test/b", "test/c"}
	for _, name := range files {
		afero.WriteFile(Fs, name, []byte(""), 0644)
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := ExpandPaths(".", test.in)
			// Sort both slices for reproducibility
			sort.Strings(actual)
			sort.Strings(test.out)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.out, actual)
		})
	}
}
