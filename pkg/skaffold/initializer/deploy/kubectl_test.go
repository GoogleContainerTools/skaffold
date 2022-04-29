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

package deploy

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateKubectlPipeline(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)

	tmpDir.Write("deployment.yaml", `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - name: getting-started
    image: gcr.io/k8s-skaffold/skaffold-example
`)
	filename := tmpDir.Path("deployment.yaml")

	k := newKubectlInitializer([]string{filename})

	expectedConfig := latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: []string{filename},
			},
		},
	}
	deployConfig, profiles := k.DeployConfig()
	testutil.CheckDeepEqual(t, expectedConfig, deployConfig)
	if profiles != nil {
		t.Errorf("generated profiles should be nil, but got: %+v\n", profiles)
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
			tmpDir := t.NewTempDir().
				Write("deployment.yaml", test.contents)

			images, err := kubernetes.ParseImagesFromKubernetesYaml(tmpDir.Path("deployment.yaml"))

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.images, images)
		})
	}
}

func TestIsKubernetesManifest(t *testing.T) {
	tests := []struct {
		description string
		filename    string
		content     string
		expected    bool
	}{
		{
			description: "valid k8s yaml filename format",
			filename:    "test1.yaml",
			content:     "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\n",
			expected:    true,
		},
		{
			description: "valid k8s json filename format",
			filename:    "test1.json",
			content:     `{"apiVersion":"v1","kind":"Service","metadata":{"name": "test"}}`,
			expected:    true,
		},
		{
			description: "valid k8s yaml filename format",
			filename:    "test1.yml",
			content:     "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\n",
			expected:    true,
		},
		{
			description: "invalid k8s yaml",
			filename:    "test1.yaml",
			content:     "key: value",
			expected:    false,
		},
		{
			description: "invalid k8s json",
			filename:    "test1.json",
			content:     `{}`,
			expected:    false,
		},
		{
			description: "invalid k8s yml",
			filename:    "test1.yml",
			content:     "key: value",
			expected:    false,
		},
		{
			description: "invalid file",
			filename:    "some.config",
			content:     "",
			expected:    false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Write(test.filename, test.content).Chdir()

			supported := kubernetes.IsKubernetesManifest(test.filename)

			t.CheckDeepEqual(test.expected, supported)
		})
	}
}
