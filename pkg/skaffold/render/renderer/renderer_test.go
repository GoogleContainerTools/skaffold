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
	"errors"
	"fmt"
	"path/filepath"
	"testing"

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
			description: "single manifests, no hydration rule",
			renderConfig: &latestV2.RenderConfig{
				Generate: &latestV2.Generate{Manifests: []string{"pod.yaml"}},
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
			description:     "manifests not given.",
			renderConfig:    &latestV2.RenderConfig{},
			originalKptfile: initKptfile,
			updatedKptfile: `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
pipeline: {}
`,
		},
		{
			description: "single manifests with validation rule.",
			renderConfig: &latestV2.RenderConfig{
				Generate: &latestV2.Generate{Manifests: []string{"pod.yaml"}},
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
			description: "Validation rule needs to be updated.",
			renderConfig: &latestV2.RenderConfig{
				Generate: &latestV2.Generate{Manifests: []string{"pod.yaml"}},
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
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			r, err := NewSkaffoldRenderer(test.renderConfig, "")
			t.CheckNoError(err)
			fakeCmd := testutil.CmdRunOut(fmt.Sprintf("kpt pkg init %v", DefaultHydrationDir), "")
			t.Override(&util.DefaultExecCommand, fakeCmd)
			t.NewTempDir().
				Write("pod.yaml", podYaml).
				Write(filepath.Join(DefaultHydrationDir, kptfile.KptFileName), test.originalKptfile).
				Touch("empty.ignored").
				Chdir()

			var b bytes.Buffer
			err = r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}})
			t.CheckNoError(err)
			t.CheckFileExistAndContent(filepath.Join(DefaultHydrationDir, dryFileName), []byte(labeledPodYaml))
			t.CheckFileExistAndContent(filepath.Join(DefaultHydrationDir, kptfile.KptFileName), []byte(test.updatedKptfile))
		})
	}
}
func TestRender_UserErr(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		r, err := NewSkaffoldRenderer(&latestV2.RenderConfig{
			Generate: &latestV2.Generate{Manifests: []string{"pod.yaml"}},
			Validate: &[]latestV2.Validator{{Name: "kubeval"}},
		}, "")
		t.CheckNoError(err)
		fakeCmd := testutil.CmdRunOutErr(fmt.Sprintf("kpt pkg init %v", DefaultHydrationDir), "",
			errors.New("fake err"))
		t.Override(&util.DefaultExecCommand, fakeCmd)
		err = r.Render(context.Background(), &bytes.Buffer{}, []graph.Artifact{{ImageName: "leeroy-web",
			Tag: "leeroy-web:v1"}})
		t.CheckContains("please manually run `kpt pkg init", err.Error())
	})
}
