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

package logfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCreate(t *testing.T) {
	var tests = []struct {
		description  string
		path         []string
		expectedName string
	}{
		{
			description:  "create file",
			path:         []string{"logs.txt"},
			expectedName: "logs.txt",
		},
		{
			description:  "create file in folder",
			path:         []string{"build", "logs.txt"},
			expectedName: filepath.Join("build", "logs.txt"),
		},
		{
			description:  "escape name",
			path:         []string{"a*name.txt"},
			expectedName: "a-name.txt",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			file, err := Create(test.path...)
			defer func() {
				file.Close()
				os.Remove(file.Name())
			}()

			t.CheckNoError(err)
			t.CheckDeepEqual(filepath.Join(os.TempDir(), "skaffold", test.expectedName), file.Name())
		})
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{name: "img", expected: "img"},
		{name: "log.txt", expected: "log.txt"},
		{name: "project/img", expected: "project-img"},
		{name: "gcr.io/project/img", expected: "gcr.io-project-img"},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			escaped := escape(test.name)

			t.CheckDeepEqual(test.expected, escaped)
		})
	}
}
