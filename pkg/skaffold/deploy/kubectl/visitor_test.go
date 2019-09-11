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

package kubectl

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

// dummyReplacer is a dummy which replaces "replace-key" with value "replaced"
// if the manifest contains a key specified in ObjIgnorer.
type dummyReplacer struct {
	m Ignorer
}

func (r *dummyReplacer) Matches(key string) bool {
	return key == "replace-key"
}

func (r *dummyReplacer) NewValue(old interface{}) (bool, interface{}) {
	return true, "replaced"
}

func (r *dummyReplacer) ObjIgnorer() Ignorer {
	return r.m
}

// mockIgnorer matches objects with ignore-key present with value in ignoreValues.
type mockIgnorer struct {
	ignoreValues []string
}

func (m mockIgnorer) Ignore(value interface{}) bool {
	for _, v := range m.ignoreValues {
		if v == value {
			return true
		}
	}
	return false
}

func (m mockIgnorer) MatchesKey(key string) bool {
	return key == "ignore-key"
}

func TestVisitReplaced(t *testing.T) {
	tests := []struct {
		description string
		matchValues []string
		manifests   ManifestList
		expected    ManifestList
	}{
		{
			description: "single manifest in the list with ignored value and string value",
			manifests: ManifestList{[]byte(`
ignore-key: ignore-value
replace-key: not-replaced`)},
			matchValues: []string{"ignore-value"},
			expected: ManifestList{[]byte(`
ignore-key: ignore-value
replace-key: not-replaced`)},
		},
		{
			description: "single manifest in the list with ignored value but not ignored",
			manifests: ManifestList{[]byte(`
ignore-key: something-else
replace-key: not-replaced`)},
			matchValues: []string{"ignore-value"},
			expected: ManifestList{[]byte(`
ignore-key: something-else
replace-key: replaced`)},
		},
		{
			description: "single manifest in the list with no ignore key but replace-key",
			manifests: ManifestList{[]byte(`
replace-key: not-replaced`)},
			matchValues: []string{"ignore-key"},
			expected: ManifestList{[]byte(`
replace-key: replaced`)},
		},
		{
			description: "multiple manifest in the list with ignored value and string value",
			manifests: ManifestList{[]byte(`
ignore-key: ignore-value-1
replace-key: not-replaced`),
				[]byte(`
ignore-key: ignore-value-2
replace-key: not-replaced`)},
			matchValues: []string{"ignore-value-1", "ignore-value-2"},
			expected: ManifestList{[]byte(`
ignore-key: ignore-value-1
replace-key: not-replaced`),
				[]byte(`
ignore-key: ignore-value-2
replace-key: not-replaced`)},
		},
		{
			description: "multiple manifest in the list with ignored value matching one obj",
			manifests: ManifestList{[]byte(`
ignore-key: not-ignored
replace-key: not-replaced`),
				[]byte(`
ignore-key: ignore-value
replace-key: not-replaced`)},
			matchValues: []string{"ignore-value"},
			expected: ManifestList{[]byte(`
ignore-key: not-ignored
replace-key: replaced`),
				[]byte(`
ignore-key: ignore-value
replace-key: not-replaced`)},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, _ := test.manifests.Visit(&dummyReplacer{mockIgnorer{test.matchValues}})
			t.CheckDeepEqual(test.expected.String(), actual.String())
		})
	}
}

func TestVisit(t *testing.T) {
	tests := []struct {
		description string
		manifests   ManifestList
		expected    ManifestList
		shouldErr   bool
	}{
		{
			description: "correct yaml",
			manifests:   ManifestList{[]byte(`test: foo`), []byte(`test: bar`)},
			expected:    ManifestList{[]byte(`test: foo`), []byte(`test: bar`)},
		},
		{
			description: "empty list",
			manifests:   ManifestList{},
			expected:    nil,
		},
		{
			description: "empty yaml in the list",
			manifests:   ManifestList{[]byte{}, []byte(`test: bar`)},
			expected:    ManifestList{[]byte(`test: bar`)},
		},
		{
			description: "incorrect yaml",
			manifests:   ManifestList{[]byte(`test:bar`)},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, err := test.manifests.Visit(&dummyReplacer{nil})
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected.String(), actual.String())
		})
	}
}

func TestVisitDifferentMatchKey(t *testing.T) {
	tests := []struct {
		description string
		manifests   ManifestList
		expected    ManifestList
	}{
		{
			description: "replace-key is repeated and before match key.",
			manifests: ManifestList{[]byte(`
repeated:
- replace-key: foo
  ignore-key: not-match
- replace-key: bar`)},
			expected: ManifestList{[]byte(`
repeated:
- ignore-key: not-match
  replace-key: replaced
- replace-key: replaced`)},
		},
		{
			description: "replace-key is not substituted in an array",
			manifests: ManifestList{[]byte(`
spec:
- replace-key
- replace-key`)},
			expected: ManifestList{[]byte(`
spec:
- replace-key
- replace-key`)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, _ := test.manifests.Visit(&dummyReplacer{mockIgnorer{[]string{"match"}}})
			t.CheckDeepEqual(test.expected.String(), actual.String())
		})
	}
}
