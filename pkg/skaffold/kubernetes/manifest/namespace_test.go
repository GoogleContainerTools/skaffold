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

package manifest

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestCollectNamespaces(t *testing.T) {
	tests := []struct {
		description string
		manifests   ManifestList
		expected    []string
	}{
		{
			description: "single Pod manifest in the list",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: []string{"test"},
		}, {
			description: "single Service manifest in the list",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Service
metadata:
  name: getting-started
  namespace: test
spec:
  type: ClusterIP
  ports:
  - port: 443
    targetPort: 8443
    protocol: TCP
  selector:
    app: getting-started
`)},
			expected: []string{"test"},
		}, {
			description: "multiple manifest in the list with different namespaces",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: foo
  namespace: test-foo
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`), []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: bar
  namespace: test-bar
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: []string{"test-bar", "test-foo"},
		}, {
			description: "multiple manifest but same namespace",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: foo
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`), []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: bar
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: []string{"test"},
		}, {
			description: "multiple manifest but no namespace",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`), []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: bar
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: []string{},
		}, {
			description: "single Pod manifest with nil namespace",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace:
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: []string{},
		}, {
			description: "single Pod manifest with empty namespace",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: ""
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: []string{},
		}, {
			description: "empty manifest",
			manifests:   ManifestList{[]byte(``)},
			expected:    []string{},
		}, {
			description: "unexpected metadata type",
			manifests:   ManifestList{[]byte(`metadata: []`)},
			expected:    []string{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, err := test.manifests.CollectNamespaces()
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestSetNamespaces(t *testing.T) {
	tests := []struct {
		description      string
		manifests        ManifestList
		expected         ManifestList
		namespace        string
		shouldErr        bool
		expectedWarnings []string
	}{
		{
			description: "single Pod manifest in the list",
			namespace:   "test",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
		}, {
			description: "multiple manifest in the list with no namespace set",
			namespace:   "test",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`), []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: bar
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: foo
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example`), []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: bar
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
		}, {
			description: "single Pod manifest with nil namespace",
			namespace:   "test",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace:
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
		}, {
			description: "single Pod manifest with empty namespace",
			namespace:   "test",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: ""
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
		}, {
			description: "empty manifest",
			namespace:   "test",
			manifests:   ManifestList{[]byte(``)},
			expected:    ManifestList{[]byte(``)},
		}, {
			description: "unexpected metadata type",
			namespace:   "test",
			manifests:   ManifestList{[]byte(`metadata: []`)},
			expected:    ManifestList{[]byte(`metadata: []`)},
		},
		{
			description: "empty metadata",
			namespace:   "test",
			manifests:   ManifestList{[]byte(``)},
			expected:    ManifestList{[]byte(``)},
		},
		{
			description: "single Pod manifest in the list with same namespace as set",
			namespace:   "test",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: test
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
		},
		{
			description: "single Pod manifest in the list with same namespace set to other",
			namespace:   "test",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: other
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: other
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			shouldErr:        false,
			expectedWarnings: []string{"a manifest already has namespace set \"other\" which conflicts with namespace on the CLI \"test\""},
		},
		{
			description: "empty namespace",
			namespace:   "",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
		},
		{
			description: "should not add namespace to any resource",
			namespace:   "test",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: getting-started
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: getting-started
`)}}, {
			description: "manifests with namespace set and empty value in CLI",
			namespace:   "",
			manifests: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: my-namespace
spec:
  containers:
    - image: gcr.io/k8s-skaffold/example
      name: example
`)},
			expected: ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: my-namespace
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			actual, err := test.manifests.SetNamespace(test.namespace,
				NewResourceSelectorLabels(TransformAllowlist, TransformDenylist))
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected.String(), actual.String(), testutil.YamlObj(t.T))
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}
