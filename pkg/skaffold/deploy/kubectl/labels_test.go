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

func TestSetLabels(t *testing.T) {
	tests := []struct {
		description string
		manifests   ManifestList
		expected    ManifestList
		labels      map[string]string
	}{
		{
			description: "Add labels when not present",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  labels:
    key1: value1
    key2: value2
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			labels: map[string]string{"key1": "value1", "key2": "value2"},
		}, {
			description: "Set labels replaces existing labels",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  labels:
    key0: value0
    key1: ignored
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  labels:
    key0: value0
    key1: value1
    key2: value2
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			labels: map[string]string{"key1": "value1", "key2": "value2"},
		}, {
			description: "no labels are set when input label is nil",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			labels: nil,
		}, {
			description: "no labels are set for kind CustomResourceDefinition",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: CustomResourceDefinition
metadata:
  name: custom
`)},
			expected: ManifestList{[]byte(`apiVersion: v1
kind: CustomResourceDefinition
metadata:
  name: custom
`)},
			labels: map[string]string{"key1": "value1"},
		}, {
			description: "no recursive metadata setting",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  validation:
    openAPIV3Schema:
      properties:
        kind:
          type: string
        metadata:
          type: object
`)},
			expected: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  validation:
    openAPIV3Schema:
      properties:
        kind:
          type: string
        metadata:
          type: object
`)},
			labels: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			resultManifest, err := test.manifests.SetLabels(test.labels)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected.String(), resultManifest.String())
		})
	}
}

func TestShouldReplaceForKind(t *testing.T) {
	tests := []struct {
		description string
		kind        string
		expect      bool
	}{
		{
			description: "shd replace for pod",
			kind:        "pod",
			expect:      true,
		},
		{
			description: "shd replace for POD (case ignore equal)",
			kind:        "POD",
			expect:      true,
		},
		{
			description: "shd replace for deployment",
			kind:        "deployment",
			expect:      true,
		},
		{
			description: "shd replace for services",
			kind:        "service",
			expect:      true,
		},
		{
			description: "shd not replace for any other kind",
			kind:        "customresourcedefinition",
		},
		{
			description: "shd not replace if kind not set",
			kind:        "",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l := newLabelsSetter(map[string]string{"test": "value"})
			l.SetKind(test.kind)
			if l.ShouldReplaceForKind() != test.expect {
				t.Errorf("expected to see %t for %s. found %t", test.expect, test.kind, !test.expect)
			}
		})
	}
}
