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

package deploy

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDefaultLabeller(t *testing.T) {
	tests := []struct {
		description string
		version     string
		expected    string
	}{
		{
			description: "version mentioned",
			version:     "1.0",
			expected:    "skaffold-1.0",
		},
		{
			description: "empty version should add postfix unknown",
			expected:    "skaffold-unknown",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&version.Get, func() *version.Info {
				return &version.Info{
					Version: test.version,
				}
			})

			l := NewLabeller(config.SkaffoldOptions{})
			labels := l.Labels()

			expected := map[string]string{
				"app.kubernetes.io/managed-by": test.expected,
				"skaffold.dev/run-id":          l.runID,
			}
			t.CheckDeepEqual(expected, labels)
		})
	}
}

func TestDefaultLabeller_TwoInstancesHaveSameRunID(t *testing.T) {
	first := NewLabeller(config.SkaffoldOptions{})
	second := NewLabeller(config.SkaffoldOptions{})

	if first.RunIDSelector() != second.RunIDSelector() {
		t.Errorf("expected the run-id to be the same for two instances")
	}
	if first.runID == "" {
		t.Error("run-id label should not be empty")
	}
}

func TestDefaultLabeller_OverrideRunId(t *testing.T) {
	labeller := NewLabeller(config.SkaffoldOptions{
		CustomLabels: []string{RunIDLabel + "=ID"},
	})

	labels := labeller.Labels()

	testutil.CheckDeepEqual(t, "ID", labels[RunIDLabel])
}

func TestLabels(t *testing.T) {
	tests := []struct {
		description    string
		options        config.SkaffoldOptions
		expectedLabels map[string]string
	}{
		{
			description:    "empty",
			options:        config.SkaffoldOptions{},
			expectedLabels: map[string]string{},
		},
		{
			description: "cleanup",
			options:     config.SkaffoldOptions{Cleanup: true},
			expectedLabels: map[string]string{
				"skaffold.dev/cleanup": "true",
			},
		},
		{
			description: "namespace",
			options:     config.SkaffoldOptions{Namespace: "NS"},
			expectedLabels: map[string]string{
				"skaffold.dev/namespace": "NS",
			},
		},
		{
			description: "profile",
			options:     config.SkaffoldOptions{Profiles: []string{"profile"}},
			expectedLabels: map[string]string{
				"skaffold.dev/profile.0": "profile",
			},
		},
		{
			description: "profiles",
			options:     config.SkaffoldOptions{Profiles: []string{"profile1", "profile2"}},
			expectedLabels: map[string]string{
				"skaffold.dev/profile.0": "profile1",
				"skaffold.dev/profile.1": "profile2",
			},
		},
		{
			description: "tail",
			options:     config.SkaffoldOptions{Tail: true},
			expectedLabels: map[string]string{
				"skaffold.dev/tail": "true",
			},
		},
		{
			description: "tail dev",
			options:     config.SkaffoldOptions{TailDev: true},
			expectedLabels: map[string]string{
				"skaffold.dev/tail": "true",
			},
		},
		{
			description: "all labels",
			options: config.SkaffoldOptions{
				Cleanup:   true,
				Namespace: "namespace",
				Profiles:  []string{"p1", "p2"},
			},
			expectedLabels: map[string]string{
				"skaffold.dev/cleanup":   "true",
				"skaffold.dev/namespace": "namespace",
				"skaffold.dev/profile.0": "p1",
				"skaffold.dev/profile.1": "p2",
			},
		},
		{
			description: "custom labels",
			options: config.SkaffoldOptions{
				Cleanup: true,
				CustomLabels: []string{
					"one=first",
					"two=second",
					"three=",
					"four",
				},
			},
			expectedLabels: map[string]string{
				"skaffold.dev/cleanup": "true",
				"one":                  "first",
				"two":                  "second",
				"three":                "",
				"four":                 "",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			l := NewLabeller(test.options)

			labels := l.Labels()

			// Ignore those two labels for this test
			delete(labels, K8sManagedByLabelKey)
			delete(labels, RunIDLabel)

			t.CheckDeepEqual(test.expectedLabels, labels)
		})
	}
}
