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

package bazel

import (
	"testing"
)

func TestDepToPath(t *testing.T) {
	var tests = []struct {
		description string
		dep         string
		expected    string
	}{
		{
			description: "top level file",
			dep:         "//:dispatcher.go",
			expected:    "dispatcher.go",
		},
		{
			description: "vendored file",
			dep:         "//vendor/github.com/gorilla/mux:mux.go",
			expected:    "vendor/github.com/gorilla/mux/mux.go",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			path := depToPath(test.dep)

			if path != test.expected {
				t.Errorf("Expected %s. Got %s", test.expected, path)
			}
		})
	}
}
