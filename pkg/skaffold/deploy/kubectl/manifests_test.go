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

package kubectl

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

const pod1 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - name: leeroy-web
    image: leeroy-web`

const pod2 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - name: leeroy-app
	image: leeroy-app`

func TestAppend(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte(pod1 + "\n---\n" + pod2))

	testutil.CheckDeepEqual(t, 2, len(manifests))
	testutil.CheckDeepEqual(t, pod1, string(manifests[0]))
	testutil.CheckDeepEqual(t, pod2, string(manifests[1]))
}

func TestAppendWithoutSeperator(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte(pod1 + "\n" + pod2))

	testutil.CheckDeepEqual(t, 2, len(manifests))
	testutil.CheckDeepEqual(t, pod1, string(manifests[0]))
	testutil.CheckDeepEqual(t, pod2, string(manifests[1]))
}
