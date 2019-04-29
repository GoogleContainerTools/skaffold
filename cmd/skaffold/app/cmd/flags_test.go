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

func TestAddFlags(t *testing.T) {
	tests := []struct {
		description   string
		annotations   map[string]string
		expectedFlags []string
	}{
		{
			description:   "not register annotations + correct annotation",
			annotations:   map[string]string{"some": "true", "test": "true"},
			expectedFlags: []string{"skip-tests"},
		},
		{
			description:   "not register annotations should return no flags",
			annotations:   map[string]string{"some": "true"},
			expectedFlags: nil,
		},
		{
			description:   "union of anotations",
			annotations:   map[string]string{"cleanup": "true", "test": "true"},
			expectedFlags: []string{"skip-tests", "cleanup", "no-prune"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			flags := getAnnotatedFlags("test", test.annotations)
			for _, f := range test.expectedFlags {
				if flags.Lookup(f) == nil {
					t.Errorf("expected flag %s to be found.", f)
				}
			}
		})
	}
}
