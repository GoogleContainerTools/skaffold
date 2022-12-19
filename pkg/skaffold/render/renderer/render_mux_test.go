/*
Copyright 2022 The Skaffold Authors

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
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

func TestRenderMux_Render(t *testing.T) {
	tests := []struct {
		name         string
		renderers    GroupRenderer
		expected     string
		expectedDeps []string
		shouldErr    bool
	}{
		{
			name: "concatenates render results with separator",
			renderers: GroupRenderer{
				Renderers: []Renderer{
					mock{configName: "config1", manifests: "manifest-1", deps: []string{"file1.txt", "file2.txt"}},
					mock{configName: "config2", manifests: "manifest-2", deps: []string{"file2.txt", "file3.txt"}},
				},
			},
			expected:     "manifest-1\n---\nmanifest-2",
			expectedDeps: []string{"file1.txt", "file2.txt", "file3.txt"},
		},
		{
			name: "returns empty string when any call fails",
			renderers: GroupRenderer{
				Renderers: []Renderer{
					mock{manifests: "manifest-1", deps: []string{"file1.txt"}},
					mock{deps: []string{"file2.txt"}, shouldErr: true}},
			},
			expectedDeps: []string{"file1.txt", "file2.txt"},
			shouldErr:    true,
		},
		{
			name: "short-circuits when first call fails",
			renderers: GroupRenderer{
				Renderers: []Renderer{
					mock{deps: []string{"file1.txt"}, shouldErr: true},
					mock{manifests: "manifest-2", deps: []string{"file2.txt"}}},
			},
			expectedDeps: []string{"file1.txt", "file2.txt"},
			shouldErr:    true,
		},
	}
	for _, tc := range tests {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			testEvent.InitializeState([]latest.Pipeline{{
				Deploy: latest.DeployConfig{},
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				}}})

			mux := NewRenderMux(tc.renderers)
			buf := &bytes.Buffer{}
			actual, err := mux.Render(context.Background(), buf, nil, true)
			actualValue := actual.String()
			t.CheckErrorAndDeepEqual(tc.shouldErr, err, tc.expected, actualValue)
			actualDeps, errD := mux.ManifestDeps()
			t.CheckNoError(errD)
			t.CheckDeepEqual(tc.expectedDeps, actualDeps)
		})
	}
}

type mock struct {
	configName string
	manifests  string
	deps       []string
	shouldErr  bool
}

func (m mock) ManifestDeps() ([]string, error) {
	return m.deps, nil
}

func (m mock) Render(context.Context, io.Writer, []graph.Artifact, bool) (manifest.ManifestListByConfig, error) {
	if m.shouldErr {
		return manifest.ManifestListByConfig{}, fmt.Errorf("render error")
	}
	manifests, err := manifest.Load(bytes.NewReader([]byte(m.manifests)))
	manifestListByConfig := manifest.NewManifestListByConfig()
	manifestListByConfig.Add(m.configName, manifests)
	return manifestListByConfig, err
}
