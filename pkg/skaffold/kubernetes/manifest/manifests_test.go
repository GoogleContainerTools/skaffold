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
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "empty", input: "", expected: []string{}},
		{name: "single doc", input: "a: b", expected: []string{"a: b\n"}}, // note lf introduced
		{name: "multiple docs", input: "a: b\n---\nc: d", expected: []string{"a: b\n", "c: d\n"}},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			result, err := Load(bytes.NewReader([]byte(test.input)))

			t.CheckError(false, err)
			t.CheckDeepEqual(len(test.expected), len(result))
			for i := range test.expected {
				t.CheckDeepEqual(test.expected[i], string(result[i]))
			}
		})
	}
}

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

const clusterRole = `aggregationRule: {}
apiVersion: v1
kind: ClusterRole`

const podUnordered = `kind: Pod
metadata:
  name: leeroy-web
apiVersion: v1
spec:
  containers:
  - name: leeroy-web
    image: leeroy-web`

const roleBinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
subjects:
- kind: ServiceAccount
  name: default
  namespace: default`

const service = `apiVersion: v1
kind: Service
metadata:
  name: my-app
spec:
  selector:
    app: my-app`

func TestEmpty(t *testing.T) {
	var manifests ManifestList

	testutil.CheckDeepEqual(t, 0, len(manifests))

	manifests.Append(nil)

	testutil.CheckDeepEqual(t, 1, len(manifests))
}

func TestAppendSingle(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte(pod1))

	testutil.CheckDeepEqual(t, 1, len(manifests))
	testutil.CheckDeepEqual(t, pod1, string(manifests[0]))
}

func TestAppendUnordered(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte(podUnordered))

	testutil.CheckDeepEqual(t, 1, len(manifests))
	testutil.CheckDeepEqual(t, podUnordered, string(manifests[0]))
}

func TestAppendWithSeparators(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte(pod1 + "\n---\n" + pod2 + "\n---\n" + podUnordered))

	testutil.CheckDeepEqual(t, 3, len(manifests))
	testutil.CheckDeepEqual(t, pod1, string(manifests[0]))
	testutil.CheckDeepEqual(t, pod2, string(manifests[1]))
	testutil.CheckDeepEqual(t, podUnordered, string(manifests[2]))
}

func TestAppendWithoutSeparators(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte(pod1 + "\n" + pod2 + "\n" + clusterRole))

	testutil.CheckDeepEqual(t, 3, len(manifests))
	testutil.CheckDeepEqual(t, pod1, string(manifests[0]))
	testutil.CheckDeepEqual(t, pod2, string(manifests[1]))
	testutil.CheckDeepEqual(t, clusterRole, string(manifests[2]))
}

func TestAppendDifferentApiVersion(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte("apiVersion: v1\napiVersion: v2"))

	testutil.CheckDeepEqual(t, 2, len(manifests))
	testutil.CheckDeepEqual(t, "apiVersion: v1", string(manifests[0]))
	testutil.CheckDeepEqual(t, "apiVersion: v2", string(manifests[1]))
}

func TestAppendServiceAndRoleBinding(t *testing.T) {
	var manifests ManifestList

	manifests.Append([]byte(roleBinding + "\n" + service))

	testutil.CheckDeepEqual(t, 2, len(manifests))
	testutil.CheckDeepEqual(t, roleBinding, string(manifests[0]))
	testutil.CheckDeepEqual(t, service, string(manifests[1]))
	testutil.CheckDeepEqual(t, manifests.String(), roleBinding+"\n---\n"+service)
}

func TestManifestListByConfigAdd(t *testing.T) {
	tests := []struct {
		description string
		mlbc        ManifestListByConfig
		config      string
		ml          ManifestList
		expected    ManifestListByConfig
	}{
		{
			description: "same config name appends to original list",
			mlbc: ManifestListByConfig{
				manifests: map[string]ManifestList{
					"config-a": {[]byte(pod1)},
				},
				configNames: []string{"config-a"},
			},
			config: "config-a",
			ml:     ManifestList{[]byte(pod2)},
			expected: ManifestListByConfig{
				manifests: map[string]ManifestList{
					"config-a": {[]byte(pod1), []byte(pod2)},
				},
				configNames: []string{"config-a"},
			},
		},
		{
			description: "different config name appends to original list",
			mlbc: ManifestListByConfig{
				manifests: map[string]ManifestList{
					"config-a": {[]byte(pod1)},
				},
				configNames: []string{"config-a"},
			},
			config: "config-b",
			ml:     ManifestList{[]byte(pod2)},
			expected: ManifestListByConfig{
				manifests: map[string]ManifestList{
					"config-a": {[]byte(pod1)},
					"config-b": {[]byte(pod2)},
				},
				configNames: []string{"config-a", "config-b"},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			test.mlbc.Add(test.config, test.ml)
			t.CheckDeepEqual(test.expected.configNames, test.mlbc.configNames)
			t.CheckDeepEqual(test.expected.manifests, test.mlbc.manifests)
		})
	}
}
