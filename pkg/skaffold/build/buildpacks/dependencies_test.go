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

package buildpacks

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependencies(t *testing.T) {
	tests := []struct {
		description string
		paths       []string
		ignore      []string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "watch everything",
			paths:       []string{"."},
			expected:    []string{"bar", filepath.FromSlash("baz/file"), "foo"},
		},
		{
			description: "watch nothing",
		},
		{
			description: "ignore some paths",
			paths:       []string{"."},
			ignore:      []string{"b*"},
			expected:    []string{"foo"},
		},
		{
			description: "glob",
			paths:       []string{"**"},
			expected:    []string{"bar", filepath.FromSlash("baz/file"), "foo"},
		},
		{
			description: "error",
			paths:       []string{"unknown"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Directory structure:
			//   foo
			//   bar
			// - baz
			//     file
			tmpDir := t.NewTempDir().
				Touch("foo", "bar", "baz/file")

			deps, err := GetDependencies(context.Background(), tmpDir.Root(), &latest.BuildpackArtifact{
				Dependencies: &latest.BuildpackDependencies{
					Paths:  test.paths,
					Ignore: test.ignore,
				},
			})

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, deps)
		})
	}
}
