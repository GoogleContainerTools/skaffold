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

package cmd

import (
	"testing"
)

func TestHasCmdAnnotation(t *testing.T) {
	tests := []struct {
		description     string
		cmd             string
		flagAnnotations []string
		expected        bool
	}{
		{
			description:     "flag has command annotations",
			cmd:             "build",
			flagAnnotations: []string{"build", "events"},
			expected:        true,
		},
		{
			description:     "flag does not have command annotations",
			cmd:             "build",
			flagAnnotations: []string{"some"},
		},
		{
			description:     "flag has all annotations",
			cmd:             "build",
			flagAnnotations: []string{"all"},
			expected:        true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.expected != hasCmdAnnotation(test.cmd, test.flagAnnotations) {
				t.Errorf("expected %t but found %t", test.expected, !test.expected)
			}
		})
	}
}
