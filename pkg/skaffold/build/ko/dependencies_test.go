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

package ko

// TODO(halvards)[09/17/2021]: Replace the latestV1 import path with the
// real schema import path once the contents of ./schema has been added to
// the real schema in pkg/skaffold/schema/latest/v1.
import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependencies(t *testing.T) {
	allFiles := []string{
		".ko.yaml",
		"cmd/foo/foo.go",
		"cmd/run.go",
		"go.mod",
		"main.go",
		"pkg/bar/bar.go",
	}
	tmpDir := testutil.NewTempDir(t).Touch(allFiles...)

	tests := []struct {
		description string
		paths       []string
		ignore      []string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "default is to watch **/*.go",
			expected: []string{
				"cmd/foo/foo.go",
				"cmd/run.go",
				"main.go",
				"pkg/bar/bar.go",
			},
		},
		{
			description: "ignore everything with nil paths",
			ignore:      []string{"."},
		},
		{
			description: "ignore everything",
			paths:       []string{"."},
			ignore:      []string{"."},
		},
		{
			description: "watch everything with empty string path",
			paths:       []string{""},
			expected:    allFiles,
		},
		{
			description: "watch everything star",
			paths:       []string{"*"},
			expected:    allFiles,
		},
		{
			description: "watch everything globstar",
			paths:       []string{"**"},
			expected:    allFiles,
		},
		{
			description: "ignore a directory",
			paths:       []string{"."},
			ignore:      []string{"cmd"},
			expected: []string{
				".ko.yaml",
				"go.mod",
				"main.go",
				"pkg/bar/bar.go",
			},
		},
		{
			description: "error",
			paths:       []string{"unknown"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			deps, err := GetDependencies(context.Background(), tmpDir.Root(), &latestV1.KoArtifact{
				Dependencies: &latestV1.KoDependencies{
					Paths:  test.paths,
					Ignore: test.ignore,
				},
			})
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, deps,
				cmpopts.AcyclicTransformer("separator", filepath.FromSlash),
			)
		})
	}
}
