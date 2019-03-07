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
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateKubeCtlPipeline(t *testing.T) {
	content := []byte(`apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - name: getting-started
    image: gcr.io/k8s-skaffold/skaffold-example
`)
	filename := createTempFileWithContents(t, "", "deployment.yaml", content)
	defer os.Remove(filename) // clean up

	expectedConfig := latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: []string{filename},
			},
		},
	}
	actual := latest.DeployConfig{}
	k, err := New([]string{filename})
	if k != nil {
		actual = k.GenerateDeployConfig()
	}
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedConfig, actual)
}

func TestParseImagesFromKubernetesYaml(t *testing.T) {
	validContent := []byte(`apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - name: getting-started
    image: gcr.io/k8s-skaffold/skaffold-example`)
	tests := []struct {
		name     string
		contents []byte
		images   []string
		err      bool
	}{
		{
			name: "incorrect k8 yaml",
			contents: []byte(`no apiVersion: t
kind: Pod`),
			images: nil,
			err:    true,
		},
		{
			name:     "correct k8 yaml",
			contents: validContent,
			images:   []string{"gcr.io/k8s-skaffold/skaffold-example"},
			err:      false,
		},
	}

	tmpDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpDir) // clean up
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpFile := createTempFileWithContents(t, tmpDir, "deployment.yaml", test.contents)
			images, err := parseImagesFromKubernetesYaml(tmpFile)
			testutil.CheckErrorAndDeepEqual(t, test.err, err, test.images, images)
		})
	}
}

func createTempFileWithContents(t *testing.T, dir string, name string, content []byte) string {
	t.Helper()
	tmpfile, err := ioutil.TempFile(dir, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	return tmpfile.Name()
}
