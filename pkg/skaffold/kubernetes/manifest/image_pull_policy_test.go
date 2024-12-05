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

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestReplaceImagePullPolicy(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: replace-imagePullPolicy
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: if-not-present
    imagePullPolicy: IfNotPresent
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: always
    imagePullPolicy: Always
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: never
    imagePullPolicy: Never
`)}

	expected := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: replace-imagePullPolicy
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: if-not-present
    imagePullPolicy: Never
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: always
    imagePullPolicy: Never
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: never
    imagePullPolicy: Never
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		resultManifest, err := manifests.ReplaceImagePullPolicy(NewResourceSelectorImagePullPolicy())
		t.CheckNoError(err)
		t.CheckDeepEqual(expected.String(), resultManifest.String(), testutil.YamlObj(t.T))
	})
}
