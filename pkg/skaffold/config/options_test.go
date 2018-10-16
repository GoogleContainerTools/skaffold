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

package config

import (
	"reflect"
	"testing"
)

func TestLabels(t *testing.T) {
	tests := []struct {
		description    string
		options        SkaffoldOptions
		expectedLabels map[string]string
	}{
		{
			description:    "empty",
			options:        SkaffoldOptions{},
			expectedLabels: map[string]string{},
		},
		{
			description:    "cleanup",
			options:        SkaffoldOptions{Cleanup: true},
			expectedLabels: map[string]string{"cleanup": "true"},
		},
		{
			description:    "namespace",
			options:        SkaffoldOptions{Namespace: "NS"},
			expectedLabels: map[string]string{"namespace": "NS"},
		},
		{
			description:    "profile",
			options:        SkaffoldOptions{Profiles: []string{"profile"}},
			expectedLabels: map[string]string{"profiles": "profile"},
		},
		{
			description:    "profiles",
			options:        SkaffoldOptions{Profiles: []string{"profile1", "profile2"}},
			expectedLabels: map[string]string{"profiles": "profile1__profile2"},
		},
		{
			description: "all labels",
			options: SkaffoldOptions{
				Cleanup:   true,
				Namespace: "namespace",
				Profiles:  []string{"p1", "p2"},
			},
			expectedLabels: map[string]string{
				"cleanup":   "true",
				"namespace": "namespace",
				"profiles":  "p1__p2",
			},
		},
		{
			description: "custom labels",
			options: SkaffoldOptions{
				Cleanup: true,
				CustomLabels: []string{
					"one=first",
					"two=second",
					"three=",
					"four",
				},
			},
			expectedLabels: map[string]string{
				"cleanup": "true",
				"one":     "first",
				"two":     "second",
				"three":   "",
				"four":    "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			labels := test.options.Labels()

			if !reflect.DeepEqual(test.expectedLabels, labels) {
				t.Errorf("Wrong labels. Expected %v. Got %v", test.expectedLabels, labels)
			}
		})
	}
}
