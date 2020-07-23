/*
Copyright 2020 The Skaffold Authors

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
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)}

	expected := ManifestList{[]byte(`
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
    name: example
`)}

	resultManifest, err := manifests.SetLabels(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
}

func TestAddLabels(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  labels:
    key0: value0
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)}

	expected := ManifestList{[]byte(`
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
    name: example
`)}

	resultManifest, err := manifests.SetLabels(map[string]string{
		"key0": "should-be-ignored",
		"key1": "value1",
		"key2": "value2",
	})

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
}

func TestSetNoLabel(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)}

	expected := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: example
`)}

	resultManifest, err := manifests.SetLabels(nil)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
}

func TestSetNoLabelWhenTypeUnexpected(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata: 3
`),
		[]byte(`
apiVersion: v1
kind: Pod
metadata:
  labels: 3
  name: getting-started
`)}

	resultManifest, err := manifests.SetLabels(map[string]string{"key0": "value0"})

	testutil.CheckErrorAndDeepEqual(t, false, err, manifests.String(), resultManifest.String())
}

func TestSetLabelInCRDSchema(t *testing.T) {
	manifests := ManifestList{[]byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mykind.mygroup.org
spec:
  group: mygroup.org
  names:
    kind: MyKind
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object`)}

	expected := ManifestList{[]byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    key0: value0
    key1: value1
  name: mykind.mygroup.org
spec:
  group: mygroup.org
  names:
    kind: MyKind
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object`)}

	resultManifest, err := manifests.SetLabels(map[string]string{
		"key0": "value0",
		"key1": "value1",
	})

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
}
