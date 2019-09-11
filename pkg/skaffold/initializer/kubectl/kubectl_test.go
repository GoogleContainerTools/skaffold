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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateKubeCtlPipeline(t *testing.T) {
	tmpDir, delete := testutil.NewTempDir(t)
	defer delete()

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

	k, err := New([]string{filename})
	if err != nil {
		t.Fatal("failed to create a pipeline")
	}

	expectedConfig := latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: []string{filename},
			},
		},
	}
	testutil.CheckDeepEqual(t, expectedConfig, k.GenerateDeployConfig())
}

func TestParseImagesFromKubernetesYaml(t *testing.T) {
	tests := []struct {
		description string
		contents    string
		images      []string
		shouldErr   bool
	}{
		{
			description: "incorrect k8 yaml",
			contents: `no apiVersion: t
kind: Pod`,
			images:    nil,
			shouldErr: true,
		},
		{
			description: "correct k8 yaml",
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
			images:    []string{},
			shouldErr: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("deployment.yaml", test.contents)

			images, err := parseImagesFromKubernetesYaml(tmpDir.Path("deployment.yaml"))

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.images, images)
		})
	}
}
