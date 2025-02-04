/*
Copyright 2025 The Skaffold Authors

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

package helm

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestNewDependencyGraph(t *testing.T) {
	tests := []struct {
		description  string
		releases     []latest.HelmRelease
		expected     map[string][]string
		shouldErr    bool
		errorMessage string
	}{
		{
			description: "simple dependency graph",
			releases: []latest.HelmRelease{
				{Name: "release1", DependsOn: []string{"release2"}},
				{Name: "release2"},
			},
			expected: map[string][]string{
				"release1": {"release2"},
				"release2": {},
			},
		},
		{
			description: "no dependencies",
			releases: []latest.HelmRelease{
				{Name: "release1"},
				{Name: "release2"},
			},
			expected: map[string][]string{
				"release1": nil,
				"release2": nil,
			},
		},
		{
			description: "multiple dependencies",
			releases: []latest.HelmRelease{
				{Name: "release1", DependsOn: []string{"release2", "release3"}},
				{Name: "release2", DependsOn: []string{"release3"}},
				{Name: "release3"},
			},
			expected: map[string][]string{
				"release1": {"release2", "release3"},
				"release2": {"release3"},
				"release3": nil,
			},
		},
		{
			description: "non-existent dependency",
			releases: []latest.HelmRelease{
				{Name: "release1", DependsOn: []string{"release3"}}, // release3 doesn't exist
				{Name: "release2"},
			},
			shouldErr:    true,
			errorMessage: "release release1 depends on non-existent release release3",
		},
		{
			description: "duplicate release names",
			releases: []latest.HelmRelease{
				{Name: "release1"},
				{Name: "release1"},
			},
			shouldErr:    true,
			errorMessage: "duplicate release name release1",
		},
		{
			description: "has cycle",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"b"}},
				{Name: "b", DependsOn: []string{"a"}},
			},
			shouldErr:    true,
			errorMessage: "cycle detected",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			graph, err := NewDependencyGraph(test.releases)

			if test.shouldErr {
				t.CheckErrorContains(test.errorMessage, err)
				return
			}
			t.CheckDeepEqual(len(test.expected), len(graph.graph))

			//nolint:gocritic
			opt := cmp.Comparer(func(x, y []string) bool {
				return slices.Equal(x, y)
			})
			if diff := cmp.Diff(test.expected, graph.graph, opt); diff != "" {
				t.Errorf("%s:got unexpected diff: %s", test.description, diff)
			}
		})
	}
}

func TestHasCycles(t *testing.T) {
	tests := []struct {
		description string
		releases    []latest.HelmRelease
		shouldErr   bool
	}{
		{
			description: "no cycles",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"b", "c"}},
				{Name: "b", DependsOn: []string{"c"}},
				{Name: "c"},
			},
			shouldErr: false,
		},
		{
			description: "simple cycle",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"b"}},
				{Name: "b", DependsOn: []string{"a"}},
			},
			shouldErr: true,
		},
		{
			description: "complex cycle",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"b"}},
				{Name: "b", DependsOn: []string{"c"}},
				{Name: "c", DependsOn: []string{"d"}},
				{Name: "d", DependsOn: []string{"b"}},
			},
			shouldErr: true,
		},
		{
			description: "empty graph",
			releases:    []latest.HelmRelease{},
			shouldErr:   false,
		},
		{
			description: "disconnected components",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"b"}},
				{Name: "b"},
				{Name: "c", DependsOn: []string{"d"}},
				{Name: "d"},
			},
			shouldErr: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := NewDependencyGraph(test.releases)

			if test.shouldErr {
				t.CheckErrorContains("cycle detected", err)
			} else {
				t.RequireNoError(err)
			}
		})
	}
}

func TestGetReleasesByLevel(t *testing.T) {
	tests := []struct {
		description string
		releases    []latest.HelmRelease
		expected    map[int][]string
	}{
		{
			description: "linear chain",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"b"}},
				{Name: "b", DependsOn: []string{"c"}},
				{Name: "c"},
			},
			expected: map[int][]string{
				0: {"c"},
				1: {"b"},
				2: {"a"},
			},
		},
		{
			description: "multiple dependencies at same level",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"c", "b"}},
				{Name: "b", DependsOn: []string{"d"}},
				{Name: "c", DependsOn: []string{"d"}},
				{Name: "d"},
			},
			expected: map[int][]string{
				0: {"d"},
				1: {"b", "c"},
				2: {"a"},
			},
		},
		{
			description: "no dependencies",
			releases: []latest.HelmRelease{
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
			},
			expected: map[int][]string{
				0: {"a", "b", "c"},
			},
		},
		{
			description: "empty graph",
			releases:    []latest.HelmRelease{},
			expected:    map[int][]string{},
		},
		{
			description: "mixed levels",
			releases: []latest.HelmRelease{
				{Name: "a", DependsOn: []string{"b", "c"}},
				{Name: "b", DependsOn: []string{"d"}},
				{Name: "c", DependsOn: []string{"d", "e"}},
				{Name: "d"},
				{Name: "e"},
			},
			expected: map[int][]string{
				0: {"d", "e"},
				1: {"b", "c"},
				2: {"a"},
			},
		},
		{
			description: "preserve order within levels",
			releases: []latest.HelmRelease{
				{Name: "a1", DependsOn: []string{"b1", "b2"}},
				{Name: "a2", DependsOn: []string{"b1", "b2"}},
				{Name: "b1", DependsOn: []string{"c1", "c2"}},
				{Name: "b2", DependsOn: []string{"c1", "c2"}},
				{Name: "c1"},
				{Name: "c2"},
			},
			expected: map[int][]string{
				0: {"c1", "c2"},
				1: {"b1", "b2"},
				2: {"a1", "a2"},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			graph, _ := NewDependencyGraph(test.releases)
			levels, err := graph.GetReleasesByLevel()
			t.RequireNoError(err)

			t.CheckDeepEqual(len(test.expected), len(levels))

			// Check that each level contains expected releases
			//nolint:gocritic
			opt := cmp.Comparer(func(x, y []string) bool {
				return slices.Equal(x, y)
			})
			if diff := cmp.Diff(test.expected, levels, opt); diff != "" {
				t.Errorf("%s: got unexpected diff (-want +got):\n%s", test.description, diff)
			}

			// Verify level assignments are correct
			for level, releases := range levels {
				for _, release := range releases {
					// Check that all dependencies are at lower levels
					for _, dep := range graph.graph[release] {
						for depLevel, depReleases := range levels {
							if slices.Contains(depReleases, dep) {
								if depLevel >= level {
									t.Errorf("dependency %s at level %d >= release %s at level %d", dep, depLevel, release, level)
								}
							}
						}
					}
				}
			}

			// Verify order preservation within each level
			// Create a map of release name to its original position
			originalOrder := make(map[string]int)
			for i, release := range test.releases {
				originalOrder[release.Name] = i
			}

			// Check that relative order is preserved within each level
			for _, releasesAtLevel := range levels {
				for i := 1; i < len(releasesAtLevel); i++ {
					prevRelease := releasesAtLevel[i-1]
					currRelease := releasesAtLevel[i]
					if originalOrder[prevRelease] > originalOrder[currRelease] {
						t.Errorf("order not preserved within level: %s (original position %d) should come after %s (original position %d)",
							prevRelease, originalOrder[prevRelease], currRelease, originalOrder[currRelease])
					}
				}
			}
		})
	}
}
