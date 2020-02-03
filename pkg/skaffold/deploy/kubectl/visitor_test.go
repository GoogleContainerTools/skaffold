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
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

type mockVisitor struct {
	visited     map[string]int
	pivotKey    string
	replaceWith interface{}
}

func (m *mockVisitor) Visit(o map[interface{}]interface{}, k interface{}, v interface{}) bool {
	s := fmt.Sprintf("%+v", v)
	if len(s) > 4 {
		s = s[:4] + "..."
	}
	m.visited[fmt.Sprintf("%v=%s", k, s)]++
	if fmt.Sprintf("%+v", o[k]) != fmt.Sprintf("%+v", v) {
		panic(fmt.Sprintf("visitor.Visit() called with o[k] != v: o[%q] != %v", k, v))
	}
	if k == m.pivotKey {
		if m.replaceWith != nil {
			o[k] = m.replaceWith
		}
		return false
	}
	return true
}

func TestVisit(t *testing.T) {
	tests := []struct {
		description       string
		pivotKey          string
		replaceWith       interface{}
		manifests         ManifestList
		expectedManifests ManifestList
		expected          []string
		shouldErr         bool
	}{
		{
			description: "correct with one level",
			manifests:   ManifestList{[]byte(`test: foo`), []byte(`test: bar`)},
			expected:    []string{"test=foo", "test=bar"},
		},
		{
			description:       "omit empty manifest",
			manifests:         ManifestList{[]byte(``), []byte(`test: bar`)},
			expectedManifests: ManifestList{[]byte(`test: bar`)},
			expected:          []string{"test=bar"},
		},
		{
			description: "nested map",
			manifests: ManifestList{[]byte(`nested:
  prop: x
test: foo`)},
			expected: []string{"test=foo", "nested=map[...", "prop=x"},
		},
		{
			description: "skip recursion at key",
			pivotKey:    "nested",
			manifests: ManifestList{[]byte(`nested:
  prop: x
test: foo`)},
			expected: []string{"test=foo", "nested=map[..."},
		},
		{
			description: "nested array and map",
			manifests: ManifestList{[]byte(`items:
- a
- 3
- name: item
  value: data
- c
test: foo`)},
			expected: []string{"test=foo", "items=[a 3...", "name=item", "value=data"},
		},
		{
			description: "replace key",
			pivotKey:    "test",
			replaceWith: "repl",
			manifests: ManifestList{[]byte(`nested:
  name: item
  test:
    sub:
      name: asdf
test: foo`), []byte(`test: bar`)},
			expectedManifests: ManifestList{[]byte(`
nested:
  name: item
  test: repl
test: repl`), []byte(`test: repl`)},
			expected: []string{"nested=map[...", "name=item", "test=map[...", "test=foo", "test=bar"},
		},
		{
			description: "invalid input",
			manifests:   ManifestList{[]byte(`test:bar`)},
			shouldErr:   true,
		},
		{
			description: "skip CRD fields",
			manifests: ManifestList{[]byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mykind.mygroup.org
spec:
  group: mygroup.org
  names:
    kind: MyKind`)},
			expected: []string{"apiVersion=apie...", "kind=Cust...", "metadata=map[...", "spec=map[..."},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			visitor := &mockVisitor{map[string]int{}, test.pivotKey, test.replaceWith}
			actual, err := test.manifests.Visit(visitor)
			expectedVisits := map[string]int{}
			for _, visit := range test.expected {
				expectedVisits[visit]++
			}
			t.CheckErrorAndDeepEqual(test.shouldErr, err, expectedVisits, visitor.visited)
			if !test.shouldErr {
				expectedManifests := test.expectedManifests
				if expectedManifests == nil {
					expectedManifests = test.manifests
				}
				t.CheckDeepEqual(expectedManifests.String(), actual.String())
			}
		})
	}
}
