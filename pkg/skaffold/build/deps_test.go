/*
Copyright 2018 The Skaffold Authors

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
package build

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func fakeDependenciesForArtifact(a *v1alpha2.Artifact) ([]string, error) {
	if a.DockerArtifact != nil {
		return []string{"Dockerfile", filepath.Join("vendor", "file")}, nil
	}
	if a.BazelArtifact != nil {
		return []string{"bazelfile", filepath.Join(".git", "config")}, nil
	}

	return nil, fmt.Errorf("undefined artifact type: %+v", a.ArtifactType)
}

func TestPaths(t *testing.T) {
	var tests = []struct {
		description string
		artifacts   []*v1alpha2.Artifact
		expected    []string
	}{
		{
			description: "correct deps from dockerfile",
			artifacts: []*v1alpha2.Artifact{
				{
					Workspace: ".",
					ArtifactType: v1alpha2.ArtifactType{
						DockerArtifact: &v1alpha2.DockerArtifact{},
					},
				},
			},
			expected: []string{"Dockerfile"},
		},
		{
			description: "correct deps from bazel",
			artifacts: []*v1alpha2.Artifact{
				{
					Workspace: "project",
					ArtifactType: v1alpha2.ArtifactType{
						BazelArtifact: &v1alpha2.BazelArtifact{},
					},
				},
			},
			expected: []string{filepath.Join("project", "bazelfile")},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			DependenciesForArtifact = fakeDependenciesForArtifact
			defer func() { DependenciesForArtifact = dependenciesForArtifact }()

			m, err := NewDependencyMap(test.artifacts)

			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, m.Paths())
		})
	}
}
