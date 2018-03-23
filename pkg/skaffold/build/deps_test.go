package build

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
)

type FakeDependencyResolver struct{}

func (f *FakeDependencyResolver) GetDependencies(a *config.Artifact) ([]string, error) {
	return []string{a.DockerfilePath}, nil
}

func TestPaths(t *testing.T) {
	var tests = []struct {
		description string
		artifacts   []*config.Artifact
		resolver    DependencyResolver
		expected    []string
	}{
		{
			description: "correct deps",
			artifacts: []*config.Artifact{
				{
					DockerfilePath: "Dockerfile",
					Workspace:      ".",
				},
			},
			resolver: &FakeDependencyResolver{},
			expected: []string{"Dockerfile"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m, err := NewDependencyMap(test.artifacts, test.resolver)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			actual := m.Paths()
			if !reflect.DeepEqual(test.expected, actual) {
				t.Errorf("%T differ.\nExpected\n%+v\nActual\n%+v", test.expected, test.expected, actual)
				return
			}
		})
	}
}

func TestArtifactsForPaths(t *testing.T) {
	var tests = []struct {
		description string
		artifacts   []*config.Artifact
		resolver    DependencyResolver
		paths       []string
		expected    []*config.Artifact
	}{
		{
			description: "correct artifacts",
			artifacts: []*config.Artifact{
				{
					DockerfilePath: "Dockerfile",
					Workspace:      ".",
				},
			},
			paths:    []string{"Dockerfile"},
			resolver: &FakeDependencyResolver{},
			expected: []*config.Artifact{
				{
					DockerfilePath: "Dockerfile",
					Workspace:      ".",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m, err := NewDependencyMap(test.artifacts, test.resolver)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			actual := m.ArtifactsForPaths(test.paths)
			if !reflect.DeepEqual(test.expected, actual) {
				t.Errorf("%T differ.\nExpected\n%+v\nActual\n%+v", test.expected, test.expected, actual)
				return
			}
		})
	}
}
