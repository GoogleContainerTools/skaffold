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

package bazel

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependencies(t *testing.T) {
	tests := []struct {
		description   string
		workspace     string
		target        string
		files         map[string]string
		expectedQuery string
		output        string
		expected      []string
		shouldErr     bool
	}{
		{
			description: "with WORKSPACE",
			workspace:   ".",
			target:      "target",
			files: map[string]string{
				"WORKSPACE": "",
				"BUILD":     "",
				"dep1":      "",
				"dep2":      "",
			},
			expectedQuery: "bazel query kind('source file', deps('target')) union buildfiles(deps('target')) --noimplicit_deps --order_output=no --output=label",
			output:        "@ignored\n//:BUILD\n//external/ignored\n\n//:dep1\n//:dep2\n",
			expected:      []string{"BUILD", "dep1", "dep2", "WORKSPACE"},
		},
		{
			description: "with parent WORKSPACE",
			workspace:   "./sub/folder",
			target:      "target2",
			files: map[string]string{
				"WORKSPACE":           "",
				"BUILD":               "",
				"sub/folder/BUILD":    "",
				"sub/folder/dep1":     "",
				"sub/folder/dep2":     "",
				"sub/folder/baz/dep3": "",
			},
			expectedQuery: "bazel query kind('source file', deps('target2')) union buildfiles(deps('target2')) --noimplicit_deps --order_output=no --output=label",
			output:        "@ignored\n//:BUILD\n//sub/folder:BUILD\n//external/ignored\n\n//sub/folder:dep1\n//sub/folder:dep2\n//sub/folder/baz:dep3\n",
			expected:      []string{filepath.Join("..", "..", "BUILD"), "BUILD", "dep1", "dep2", filepath.Join("baz", "dep3"), filepath.Join("..", "..", "WORKSPACE")},
		},
		{
			description: "without WORKSPACE",
			workspace:   ".",
			target:      "target",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				test.expectedQuery,
				test.output,
			))
			t.NewTempDir().WriteFiles(test.files).Chdir()

			deps, err := GetDependencies(context.Background(), test.workspace, &latest.BazelArtifact{
				BuildTarget: test.target,
			})

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, deps)
		})
	}
}

func TestQuery(t *testing.T) {
	query := query("//:skaffold_example.tar")

	expectedQuery := `kind('source file', deps('//:skaffold_example.tar')) union buildfiles(deps('//:skaffold_example.tar'))`
	if query != expectedQuery {
		t.Errorf("Expected [%s]. Got [%s]", expectedQuery, query)
	}
}

func TestDepToPath(t *testing.T) {
	tests := []struct {
		description string
		dep         string
		expected    string
	}{
		{
			description: "top level file",
			dep:         "//:dispatcher.go",
			expected:    "dispatcher.go",
		},
		{
			description: "vendored file",
			dep:         "//vendor/github.com/gorilla/mux:mux.go",
			expected:    "vendor/github.com/gorilla/mux/mux.go",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			path := depToPath(test.dep)

			t.CheckDeepEqual(test.expected, path)
		})
	}
}
