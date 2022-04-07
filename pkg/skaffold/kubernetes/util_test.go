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

package kubernetes

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSupportedKubernetesFormats(t *testing.T) {
	tests := []struct {
		description string
		in          string
		out         bool
	}{
		{
			description: "yaml",
			in:          "filename.yaml",
			out:         true,
		},
		{
			description: "yml",
			in:          "filename.yml",
			out:         true,
		},
		{
			description: "json",
			in:          "filename.json",
			out:         true,
		},
		{
			description: "txt",
			in:          "filename.txt",
			out:         false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := HasKubernetesFileExtension(test.in)

			t.CheckDeepEqual(test.out, actual)
		})
	}
}

func TestParseImagesFromKubernetesYaml(t *testing.T) {
	tests := []struct {
		description string
		contents    string
		images      []string
		shouldErr   bool
	}{
		{
			description: "incorrect k8s yaml",
			contents: `no apiVersion: t
		kind: Pod`,
			images:    nil,
			shouldErr: true,
		},
		{
			description: "correct k8s yaml",
			contents: `apiVersion: v1
		kind: Pod
		metadata:
		 name: getting-started
		spec:
		 containers:
		 - name: getting-started
		   image: gcr.io/k8s-skaffold/skaffold-example`,
			images:    []string{"gcr.io/k8s-skaffold/skaffold-example"},
			shouldErr: false,
		},
		{
			description: "correct rolebinding yaml with no image",
			contents: `apiVersion: rbac.authorization.k8s.io/v1
		kind: RoleBinding
		metadata:
		 name: default-admin
		 namespace: default
		roleRef:
		 apiGroup: rbac.authorization.k8s.io
		 kind: ClusterRole
		 name: admin
		subjects:
		- name: default
		 kind: ServiceAccount
		 namespace: default`,
			images:    nil,
			shouldErr: false,
		},
		{
			description: "crd",
			contents: `apiVersion: my.crd.io/v1
		kind: CustomType
		metadata:
		 name: test crd
		spec:
		 containers:
		 - name: container
		   image: gcr.io/my/image`,
			images:    []string{"gcr.io/my/image"},
			shouldErr: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			images, err := parseImagesFromKubernetesYaml(bufio.NewReader(bytes.NewBufferString(test.contents)))
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.images, images)
		})
	}
}
