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
package renderer

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
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
    name: leeroy-web
`
	// manifests with image labels
	labeledPodYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
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
		renderConfig    *latestV2.RenderConfig
		originalKptfile string
		updatedKptfile  string
	}{
		{
			description: "single manifest, no hydration rule",
			renderConfig: &latestV2.RenderConfig{
				Generate: latestV2.Generate{RawK8s: []string{"pod.yaml"}},
			},
			originalKptfile: initKptfile,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline: {}
`,
		},
		{
			description: "manifests with validation rule.",
			renderConfig: &latestV2.RenderConfig{
				Generate: latestV2.Generate{RawK8s: []string{"pod.yaml"}},
				Validate: &[]latestV2.Validator{{Name: "kubeval"}},
			},
			originalKptfile: initKptfile,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  validators:
  - image: gcr.io/kpt-fn/kubeval:v0.1
`,
		},
		{
			description: "manifests with updated validation rule.",
			renderConfig: &latestV2.RenderConfig{
				Generate: latestV2.Generate{RawK8s: []string{"pod.yaml"}},
				Validate: &[]latestV2.Validator{{Name: "kubeval"}},
			},
			originalKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  validators:
  - image: gcr.io/kpt-fn/SOME-OTHER-FUNC
`,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  validators:
  - image: gcr.io/kpt-fn/kubeval:v0.1
`,
		},
		{
			description: "manifests with transformation rule.",
			renderConfig: &latestV2.RenderConfig{
				Generate:  latestV2.Generate{RawK8s: []string{"pod.yaml"}},
				Transform: &[]latestV2.Transformer{{Name: "set-labels", ConfigMap: []string{"owner:tester"}}},
			},
			originalKptfile: initKptfile,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-labels:v0.1
    configMap:
      owner: tester
`,
		},
		{
			description: "manifests with updated transformation rule.",
			renderConfig: &latestV2.RenderConfig{
				Generate:  latestV2.Generate{RawK8s: []string{"pod.yaml"}},
				Transform: &[]latestV2.Transformer{{Name: "set-labels", ConfigMap: []string{"owner:tester"}}},
			},
			originalKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/SOME-OTHER-FUNC
`,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-labels:v0.1
    configMap:
      owner: tester
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDirObj := t.NewTempDir()
			tmpDirObj.Write("pod.yaml", podYaml).
				Write(filepath.Join(constants.DefaultHydrationDir, kptfile.KptFileName), test.originalKptfile).
				Touch("empty.ignored").
				Chdir()
			r, err := NewSkaffoldRenderer(test.renderConfig, tmpDirObj.Root(),
				filepath.Join(tmpDirObj.Root(), constants.DefaultHydrationDir))
			t.CheckNoError(err)
			t.Override(&util.DefaultExecCommand,
				testutil.CmdRun(fmt.Sprintf("kpt fn render %v",
					filepath.Join(tmpDirObj.Root(), ".kpt-pipeline"))))
			var b bytes.Buffer
			err = r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				true, "")
			t.CheckNoError(err)
			t.CheckFileExistAndContent(filepath.Join(tmpDirObj.Root(), constants.DefaultHydrationDir, dryFileName), []byte(labeledPodYaml))
			t.CheckFileExistAndContent(filepath.Join(tmpDirObj.Root(), constants.DefaultHydrationDir, kptfile.KptFileName), []byte(test.updatedKptfile))
		})
	}
}

func TestRender_StashKptinfo(t *testing.T) {
	tests := []struct {
		description     string
		originalKptfile string
		updatedKptfile  string
	}{
		{
			description:     "kpt initialized, manifests are not kpt applied before (no inventory info)",
			originalKptfile: initKptfile,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  validators:
  - image: gcr.io/kpt-fn/kubeval:v0.1
`,
		},
		{
			description: "manifests has been previously kpt applied (with inventory info)",
			originalKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
inventory:
  namespace: skaffold-test
  inventoryID: 11111
`,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline:
  validators:
  - image: gcr.io/kpt-fn/kubeval:v0.1
inventory:
  namespace: skaffold-test
  inventoryID: "11111"
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDirObj := t.NewTempDir()
			tmpDirObj.Write("pod.yaml", podYaml).
				Write(filepath.Join(constants.DefaultHydrationDir, kptfile.KptFileName), test.originalKptfile).
				Chdir()
			r, err := NewSkaffoldRenderer(&latestV2.RenderConfig{
				Generate: latestV2.Generate{RawK8s: []string{"pod.yaml"}},
				Validate: &[]latestV2.Validator{{Name: "kubeval"}}}, tmpDirObj.Root(),
				filepath.Join(tmpDirObj.Root(), constants.DefaultHydrationDir))
			t.CheckNoError(err)
			t.Override(&util.DefaultExecCommand,
				testutil.CmdRun(fmt.Sprintf("kpt fn render %v",
					filepath.Join(tmpDirObj.Root(), ".kpt-pipeline"))))
			var b bytes.Buffer
			err = r.Render(context.Background(), &b, []graph.Artifact{},
				true, "")
			t.CheckNoError(err)
			t.CheckFileExistAndContent(filepath.Join(tmpDirObj.Root(), constants.DefaultHydrationDir, kptfile.KptFileName),
				[]byte(test.updatedKptfile))
		})
	}
}
