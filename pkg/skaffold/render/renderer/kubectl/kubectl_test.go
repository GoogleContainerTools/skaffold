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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`
	// manifests with image tags
	taggedPodYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`
)

func TestRender(t *testing.T) {
	tests := []struct {
		description  string
		renderConfig *latest.ManifestConfig
		labels       map[string]string
		expected     string
	}{
		{
			description: "single manifest with no labels",
			renderConfig: &latest.RenderConfig{
				Generate: latest.Generate{RawK8s: []string{"pod.yaml"}},
			},
			expected: taggedPodYaml,
		},
		{
			description: "single manifest with labels",
			renderConfig: &latest.RenderConfig{
				Generate: latest.Generate{RawK8s: []string{"pod.yaml"}},
			},
			labels:   map[string]string{"run.id": "test"},
			expected: labeledPodYaml,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDirObj := t.NewTempDir()
			tmpDirObj.Write("pod.yaml", podYaml).
				Touch("empty.ignored").
				Chdir()
			mockCfg := mockConfig{workingDir: tmpDirObj.Root()}
			r, err := New(mockCfg, test.renderConfig, test.labels)
			t.CheckNoError(err)
			var b bytes.Buffer
			manifestList, errR := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				true)
			t.CheckNoError(errR)
			t.CheckDeepEqual(test.expected, manifestList.String())
		})
	}
}

type mockConfig struct {
	workingDir string
}

func (mc mockConfig) GetWorkingDir() string                       { return mc.workingDir }
func (mc mockConfig) TransformAllowList() []latest.ResourceFilter { return nil }
func (mc mockConfig) TransformDenyList() []latest.ResourceFilter  { return nil }
func (mc mockConfig) TransformRulesFile() string                  { return "" }
