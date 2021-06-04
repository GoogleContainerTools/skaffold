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

func TestRender_StoredInCache(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		r := NewSkaffoldRenderer(&latestV2.RenderConfig{Generate: &latestV2.Generate{
			Manifests: []string{"pod.yaml"}}}, "")
		fakeCmd := testutil.CmdRunOut(fmt.Sprintf("kpt pkg init %v", DefaultHydrationDir), "")
		t.Override(&util.DefaultExecCommand, fakeCmd)
		t.NewTempDir().
			Write("pod.yaml", podYaml).
			Write(filepath.Join(DefaultHydrationDir, kptfile.KptFileName), initKptfile).
			Touch("empty.ignored").
			Chdir()

		var b bytes.Buffer
		err := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}})
		t.CheckNoError(err)
		t.CheckFileExistAndContent(filepath.Join(DefaultHydrationDir, dryFileName), []byte(labeledPodYaml))
	})
}
