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

package kpt

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
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
    name: leeroy-web
`
	// manifests with image labels
	labeledPodYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
  namespace: default
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web
`
	initKptfile = `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
`
)

func TestRender(t *testing.T) {
	tests := []struct {
		description     string
		renderConfig    latest.RenderConfig
		config          *runcontext.RunContext
		originalKptfile string
		updatedKptfile  string
	}{
		{
			description: "single manifest",
			renderConfig: latest.RenderConfig{
				Generate: latest.Generate{},
			},
			originalKptfile: initKptfile,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDirObj := t.NewTempDir()
			tmpDirObj.Write("pod.yaml", podYaml).
				Write(filepath.Join(constants.DefaultHydrationDir, kptfile.KptFileName), test.originalKptfile).
				Touch("empty.ignored").
				Chdir()
			mockCfg := render.MockConfig{
				WorkingDir: tmpDirObj.Root(),
			}
			test.renderConfig.Kpt = []string{filepath.Join(tmpDirObj.Root(), constants.DefaultHydrationDir)}
			r, err := New(mockCfg, test.renderConfig, filepath.Join(tmpDirObj.Root(), constants.DefaultHydrationDir), map[string]string{}, "default", "", nil)
			t.CheckNoError(err)
			t.Override(&util.DefaultExecCommand,
				testutil.CmdRunOut(fmt.Sprintf("kpt fn render %v -o unwrap",
					filepath.Join(tmpDirObj.Root(), ".kpt-pipeline")), podYaml))
			var b bytes.Buffer
			manifests, err := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				false)
			t.CheckNoError(err)
			s := manifests.String() + "\n"
			t.CheckDeepEqual(s, labeledPodYaml, testutil.YamlObj(t.T))
		})
	}
}
