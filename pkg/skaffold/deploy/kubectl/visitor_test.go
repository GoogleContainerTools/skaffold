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
// if the manifest contains a key specified in ObjMatcher.
type dummyReplacer struct {
	m Matcher
}

func (r *dummyReplacer) Matches(key string) bool {
	return key == "replace-key"
}

func (r *dummyReplacer) NewValue(old interface{}) (bool, interface{}) {
	return true, "replaced"
}

func (r *dummyReplacer) ObjMatcher() Matcher {
	return r.m
}

// mockMatcher matches objects with match-key present with value in matchValues.
type mockMatcher struct {
	matchValues []string
}

func (m mockMatcher) Matches(value interface{}) bool {
	for _, v := range m.matchValues {
		if v == value {
			return true
		}
	}
	return false
}

func (m mockMatcher) IsMatchKey(key string) bool {
	return key == "match-key"
}

func TestVisitReplaced(t *testing.T) {
	tests := []struct {
		description string
		matchValues []string
		manifests   ManifestList
		expected    ManifestList
	}{
		{
			description: "single manifest in the list with matched key and string value",
			manifests: ManifestList{[]byte(`
match-key: match-value
replace-key: not-replaced`)},
			matchValues: []string{"match-value"},
			expected: ManifestList{[]byte(`
match-key: match-value
replace-key: replaced`)},
		},
		{
			description: "single manifest in the list with matched key but incorrect value",
			manifests: ManifestList{[]byte(`
match-key: something-else
replace-key: not-replaced`)},
			matchValues: []string{"match-value"},
			expected: ManifestList{[]byte(`
match-key: something-else
replace-key: not-replaced`)},
		},
		{
			description: "single manifest in the list with no match key but replace-key",
			manifests: ManifestList{[]byte(`
replace-key: not-replaced`)},
			matchValues: []string{"match-key"},
			expected: ManifestList{[]byte(`
replace-key: replaced`)},
		},
		{
			description: "multiple manifest in the list with matched key and string value",
			manifests: ManifestList{[]byte(`
match-key: match-value-1
replace-key: not-replaced`),
				[]byte(`
match-key: match-value-2
replace-key: not-replaced`)},
			matchValues: []string{"match-value-1", "match-value-2"},
			expected: ManifestList{[]byte(`
match-key: match-value-1
replace-key: replaced`),
				[]byte(`
match-key: match-value-2
replace-key: replaced`)},
		},
		{
			description: "multiple manifest in the list with matched key matching one obj",
			manifests: ManifestList{[]byte(`
match-key: not-matched
replace-key: not-replaced`),
				[]byte(`
match-key: match-value
replace-key: not-replaced`)},
			matchValues: []string{"match-value"},
			expected: ManifestList{[]byte(`
match-key: not-matched
replace-key: not-replaced`),
				[]byte(`
match-key: match-value
replace-key: replaced`)},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, _ := test.manifests.Visit(&dummyReplacer{mockMatcher{test.matchValues}})
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
  match-key: match
- replace-key: bar`)},
			expected: ManifestList{[]byte(`
repeated:
- match-key: match
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
			actual, _ := test.manifests.Visit(&dummyReplacer{mockMatcher{[]string{"match"}}})
			t.CheckDeepEqual(test.expected.String(), actual.String())
		})
	}
}
