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

package custom

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependenciesDockerfile(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	// Directory structure:
	//   foo
	//   bar
	// - baz
	//     file
	//   Dockerfile
	tmpDir.Write("foo", "")
	tmpDir.Write("bar", "")
	tmpDir.Mkdir("baz")
	tmpDir.Write("baz/file", "")
	tmpDir.Write("Dockerfile", "FROM scratch \n ARG file \n COPY $file baz/file .")

	customArtifact := &latest.CustomArtifact{
		Dependencies: &latest.CustomDependencies{
			Dockerfile: &latest.DockerfileDependency{
				Path: "Dockerfile",
				BuildArgs: map[string]*string{
					"file": stringPointer("foo"),
				},
			},
		},
	}

	expected := []string{"Dockerfile", filepath.FromSlash("baz/file"), "foo"}
	deps, err := GetDependencies(context.Background(), tmpDir.Root(), customArtifact, nil)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, deps)
}

func TestGetDependenciesCommand(t *testing.T) {
	reset := testutil.Override(t, &util.DefaultExecCommand, testutil.NewFakeCmd(t).WithRunOut(
		"echo [\"file1\",\"file2\",\"file3\"]",
		"[\"file1\",\"file2\",\"file3\"]",
	))
	defer reset()

	customArtifact := &latest.CustomArtifact{
		Dependencies: &latest.CustomDependencies{
			Command: "echo [\"file1\",\"file2\",\"file3\"]",
		},
	}

	expected := []string{"file1", "file2", "file3"}
	deps, err := GetDependencies(context.Background(), "", customArtifact, nil)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, deps)
}

func TestGetDependenciesPaths(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	// Directory structure:
	//   foo
	//   bar
	// - baz
	//     file
	tmpDir.Write("foo", "")
	tmpDir.Write("bar", "")
	tmpDir.Mkdir("baz")
	tmpDir.Write("baz/file", "")

	tests := []struct {
		description string
		ignore      []string
		paths       []string
		expected    []string
	}{
		{
			description: "watch everything",
			paths:       []string{"."},
			expected:    []string{"bar", filepath.FromSlash("baz/file"), "foo"},
		}, {
			description: "watch nothing",
		}, {
			description: "ignore some paths",
			paths:       []string{"."},
			ignore:      []string{"b*"},
			expected:    []string{"foo"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			deps, err := GetDependencies(context.Background(), tmpDir.Root(), &latest.CustomArtifact{
				Dependencies: &latest.CustomDependencies{
					Paths:  test.paths,
					Ignore: test.ignore,
				},
			}, nil)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, deps)
		})
	}

}

func stringPointer(s string) *string {
	return &s
}
