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

package kubectl

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	// Raw manifests
	podYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - image: leeroy-web
    name: leeroy-web`
	// manifests with image tags and label
	labeledPodYaml = `apiVersion: v1
kind: Pod
metadata:
  labels:
    run.id: test
  name: leeroy-web
  namespace: default
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`
	// manifests with image tags
	taggedPodYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
  namespace: default
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`
)

func TestRender(t *testing.T) {
	tests := []struct {
		description  string
		renderConfig latest.RenderConfig
		labels       map[string]string
		expected     string
		cmpOptions   cmp.Options
	}{
		{
			description: "single manifest with no labels",
			renderConfig: latest.RenderConfig{
				Generate: latest.Generate{RawK8s: []string{"pod.yaml"}},
			},
			expected:   taggedPodYaml,
			cmpOptions: []cmp.Option{testutil.YamlObj(t)},
		},
		{
			description: "single manifest with labels",
			renderConfig: latest.RenderConfig{
				Generate: latest.Generate{RawK8s: []string{"pod.yaml"}},
			},
			labels:     map[string]string{"run.id": "test"},
			expected:   labeledPodYaml,
			cmpOptions: []cmp.Option{testutil.YamlObj(t)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDirObj := t.NewTempDir()
			tmpDirObj.Write("pod.yaml", podYaml).
				Touch("empty.ignored").
				Chdir()
			mockCfg := render.MockConfig{WorkingDir: tmpDirObj.Root()}
			r, err := New(mockCfg, test.renderConfig, test.labels, "default", "", nil)
			t.CheckNoError(err)
			var b bytes.Buffer
			manifestList, errR := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				false)
			t.CheckNoError(errR)
			t.CheckDeepEqual(test.expected, manifestList.String(), test.cmpOptions)
		})
	}
}

func TestDependencies(t *testing.T) {
	tests := []struct {
		description string
		manifests   []string
		expected    []string
	}{
		{
			description: "no manifest",
			manifests:   []string(nil),
			expected:    []string(nil),
		},
		{
			description: "missing manifest file",
			manifests:   []string{"missing.yaml"},
			expected:    []string(nil),
		},
		{
			description: "ignore non-manifest",
			manifests:   []string{"*.ignored"},
			expected:    []string(nil),
		},
		{
			description: "single manifest",
			manifests:   []string{"deployment.yaml"},
			expected:    []string{"deployment.yaml"},
		},
		{
			description: "keep manifests order",
			manifests:   []string{"01_name.yaml", "00_service.yaml"},
			expected:    []string{"01_name.yaml", "00_service.yaml"},
		},
		{
			description: "sort children",
			manifests:   []string{"01/*.yaml", "00/*.yaml"},
			expected:    []string{filepath.Join("01", "a.yaml"), filepath.Join("01", "b.yaml"), filepath.Join("00", "a.yaml"), filepath.Join("00", "b.yaml")},
		},
		{
			description: "http manifest",
			manifests:   []string{"deployment.yaml", "http://remote.yaml"},
			expected:    []string{"deployment.yaml"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			tmpDir.Touch("deployment.yaml", "01_name.yaml", "00_service.yaml", "empty.ignored").
				Touch("01/a.yaml", "01/b.yaml").
				Touch("00/b.yaml", "00/a.yaml").
				Chdir()

			mockCfg := render.MockConfig{WorkingDir: tmpDir.Root()}
			rCfg := latest.RenderConfig{
				Generate: latest.Generate{RawK8s: test.manifests},
			}
			r, err := New(mockCfg, rCfg, map[string]string{}, "default", "", nil)
			t.CheckNoError(err)

			dependencies, err := r.ManifestDeps()
			t.CheckNoError(err)
			if len(dependencies) == 0 {
				t.CheckDeepEqual(test.expected, dependencies)
			} else {
				expected := make([]string, len(test.expected))
				for i, p := range test.expected {
					expected[i] = filepath.Join(tmpDir.Root(), p)
				}
				t.CheckDeepEqual(expected, dependencies)
			}
		})
	}
}
