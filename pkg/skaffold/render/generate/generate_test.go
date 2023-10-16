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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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
    image: leeroy-web`

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
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		description         string
		generateConfig      latest.Generate
		expected            manifest.ManifestList
		commands            util.Command
		useKubectlKustomize bool
	}{
		{
			description: "render raw manifests",
			generateConfig: latest.Generate{
				RawK8s: []string{"pod.yaml"},
			},
			expected: manifest.ManifestList{[]byte(podYaml)},
		},
		{
			description: "render glob raw manifests",
			generateConfig: latest.Generate{
				RawK8s: []string{"*.yaml"},
			},
			expected: manifest.ManifestList{[]byte(podYaml), []byte(podsYaml)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&KubectlVersionCheck, func(*kubectl.CLI) bool {
				return test.useKubectlKustomize
			})
			t.Override(&KustomizeBinaryCheck, func() bool {
				return test.useKubectlKustomize
			})
			t.NewTempDir().
				Write("pod.yaml", podYaml).
				Write("pods.yaml", podsYaml).
				Touch("empty.ignored").
				Chdir()

			g := NewGenerator(".", test.generateConfig, "")
			var output bytes.Buffer
			actual, err := g.Generate(context.Background(), &output)
			t.CheckNoError(err)
			t.CheckDeepEqual(actual.String(), test.expected.String())
		})
	}
}

func TestGenerateFromURLManifest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, podYaml)
	}))

	defer ts.Close()
	defer os.RemoveAll(manifest.ManifestTmpDir)
	g := NewGenerator(".", latest.Generate{
		RawK8s: []string{ts.URL},
	}, "")
	var output bytes.Buffer
	actual, err := g.Generate(context.Background(), &output)
	testutil.Run(t, "", func(t *testutil.T) {
		t.CheckNoError(err)
		manifestList := manifest.ManifestList{[]byte(podYaml)}
		t.CheckDeepEqual(actual.String(), manifestList.String())
	})
}

func TestManifestDeps(t *testing.T) {
	tests := []struct {
		description    string
		generateConfig latest.Generate
		expected       []string
	}{
		{
			description: "rawYaml dir",
			generateConfig: latest.Generate{
				RawK8s: []string{"rawYaml-sample"},
			},
			expected: []string{"rawYaml-sample/pod.yaml", "rawYaml-sample/pods2.yaml"},
		},
		{
			description: "rawYaml specific",
			generateConfig: latest.Generate{
				RawK8s: []string{"rawYaml-sample/pod.yaml"},
			},
			expected: []string{"rawYaml-sample/pod.yaml"},
		},
		{
			description: "multi manifest, mixed dir and file",
			generateConfig: latest.Generate{
				RawK8s: []string{"rawYaml-sample"},
			},
			expected: []string{
				"rawYaml-sample/pod.yaml",
				"rawYaml-sample/pods2.yaml",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			tmpDir.Write("rawYaml-sample/pod.yaml", podYaml).
				Write("rawYaml-sample/pods2.yaml", podsYaml).
				Write("rawYaml-sample/irrelevant.txt", "").
				Touch("empty.ignored").
				Chdir()
			expectedPaths := []string{}
			for _, p := range test.expected {
				expectedPaths = append(expectedPaths, filepath.Join(tmpDir.Root(), p))
			}
			g := Generator{config: test.generateConfig, workingDir: tmpDir.Root()}
			actual, err := g.ManifestDeps()
			t.CheckNoError(err)
			t.CheckDeepEqual(expectedPaths, actual)
		})
	}
}
