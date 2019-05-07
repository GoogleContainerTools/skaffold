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
	var tests = []struct {
		description string
		manifests   ManifestList
		expected    ManifestList
		labels      map[string]string
	}{
		{
			description: "set labels when no labels are present",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  labels:
    key1: value1
    key2: value2
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`)},
			labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}, {
			description: "add labels to existing labels",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  labels:
    key0: value0
    key1: ignored
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
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
    name: example`)},
			labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}, {
			description: "set no labels",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`)},
			labels: nil,
		}, {
			description: "adds labels recursively",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Deployment
metadata:
  name: getting-started
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: skaffold-helm
        release: skaffold-helm`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Deployment
metadata:
  labels:
    key1: value1
  name: getting-started
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: skaffold-helm
        key1: value1
        release: skaffold-helm`)},
			labels: map[string]string{
				"key1": "value1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := test.manifests.SetLabels(test.labels)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected.String(), actual.String())
		})
	}

}
