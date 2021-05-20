/*
Copyright 2021 The Skaffold Authors

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
package generate

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	// Test file under <tmp>/pod.yaml
	podYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - name: leeroy-web
    image: leeroy-web
`

	// Test file under <tmp>/pods.yaml. This file contains multiple config object.
	podsYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web2
spec:
  containers:
  - name: leeroy-web2
    image: leeroy-web2
---
apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web3
spec:
  containers:
  - name: leeroy-web3
    image: leeroy-web3
`

	// Test file under <tmp>/base/patch.yaml
	patchYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kustomize-test
spec:
  template:
    spec:
      containers:
      - name: kustomize-test
        image: index.docker.io/library/busybox
        command:
          - sleep
          - "3600"
`
	// Test file under <tmp>/base/deployment.yaml
	kustomizeDeploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kustomize-test
  labels:
    app: kustomize-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kustomize-test
  template:
    metadata:
      labels:
        app: kustomize-test
    spec:
      containers:
      - name: kustomize-test
        image: not/a/valid/image
`
	// The kustomize build result from kustomizeYaml file.
	kustomizePatchedOutput = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: kustomize-test
  name: kustomize-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kustomize-test
  template:
    metadata:
      labels:
        app: kustomize-test
    spec:
      containers:
      - command:
        - sleep
        - "3600"
        image: index.docker.io/library/busybox
        name: kustomize-test
`
	// Test file under <tmp>/base/kustomization.yaml
	kustomizeYaml = `resources:
  - deployment.yaml
patches:
  - patch.yaml
`
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		description    string
		generateConfig latestV2.Generate
		expected       manifest.ManifestList
	}{
		{
			description: "render raw manifests",
			generateConfig: latestV2.Generate{
				Manifests: []string{"pod.yaml"},
			},
			expected: manifest.ManifestList{[]byte(podYaml)},
		},
		{
			description: "render glob raw manifests",
			generateConfig: latestV2.Generate{
				Manifests: []string{"*.yaml"},
			},
			expected: manifest.ManifestList{[]byte(podYaml), []byte(podsYaml)},
		},
		{
			description: "render kustomize manifests",
			generateConfig: latestV2.Generate{
				Manifests: []string{"base"},
			},
			expected: manifest.ManifestList{[]byte(kustomizePatchedOutput)},
		},

		{
			description: "render mixed raw and kustomize manifests",
			generateConfig: latestV2.Generate{
				Manifests: []string{"*"},
			},
			expected: manifest.ManifestList{[]byte(kustomizePatchedOutput), []byte(podYaml), []byte(podsYaml)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeCmd := testutil.CmdRunOut("kustomize build base", kustomizePatchedOutput)
			t.Override(&util.DefaultExecCommand, fakeCmd)

			t.NewTempDir().
				Write("pod.yaml", podYaml).
				Write("pods.yaml", podsYaml).
				Write("base/kustomization.yaml", kustomizeYaml).
				Write("base/patch.yaml", patchYaml).
				Write("base/deployment.yaml", kustomizeDeploymentYaml).
				Touch("empty.ignored").
				Chdir()

			g := NewGenerator(".", test.generateConfig)
			actual, err := g.Generate(context.Background())
			t.CheckNoError(err)
			t.CheckDeepEqual(actual.String(), test.expected.String())
		})
	}
}
