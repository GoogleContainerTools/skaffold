/*
Copyright 2018 Google LLC

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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/google/go-cmp/cmp"
)

type FakeDependencyResolver struct {
	deps []string
}

func (f *FakeDependencyResolver) GetDependencies(a *v1alpha2.Artifact) ([]string, error) {
	return f.deps, nil
}

func TestPaths(t *testing.T) {
	var tests = []struct {
		description    string
		artifacts      []*v1alpha2.Artifact
		dockerResolver DependencyResolver
		bazelResolver  DependencyResolver
		expected       []string
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
			dockerResolver: &FakeDependencyResolver{deps: []string{"Dockerfile"}},
			expected:       []string{"Dockerfile"},
		},
		{
			description: "correct deps from bazel",
			artifacts: []*v1alpha2.Artifact{
				{
					Workspace: ".",
					ArtifactType: v1alpha2.ArtifactType{
						BazelArtifact: &v1alpha2.BazelArtifact{},
					},
				},
			},
			bazelResolver: &FakeDependencyResolver{deps: []string{"bazelfile"}},
			expected:      []string{"bazelfile"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			oldDockerResolver := DefaultDockerfileDepResolver
			oldBazelResolver := DefaultBazelDepResolver

			DefaultDockerfileDepResolver = test.dockerResolver
			DefaultBazelDepResolver = test.bazelResolver

			m, err := NewDependencyMap(test.artifacts)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			DefaultDockerfileDepResolver = oldDockerResolver
			DefaultBazelDepResolver = oldBazelResolver

			if diff := cmp.Diff(test.expected, m.Paths()); diff != "" {
				t.Errorf("%T differ.\nExpected\n%+v\nActual\n%+v", test.expected, test.expected, m.Paths())
				return
			}

		})
	}
}
