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

package helm

import (
	"slices"
	"testing"

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
				{Name: "release2", DependsOn: []string{}},
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
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			graph, err := NewDependencyGraph(test.releases)

			if test.shouldErr {
				t.CheckErrorContains(test.errorMessage, err)
				return
			}
			t.CheckDeepEqual(len(test.expected), len(graph.graph))

			for release, deps := range test.expected {
				actualDeps, exists := graph.graph[release]
				if !exists {
					t.Errorf("missing release %s in graph", release)
					continue
				}

				if len(deps) != len(actualDeps) {
					t.Errorf("expected %d dependencies for %s, got %d", len(deps), release, len(actualDeps))
					continue
				}

				// Check all expected dependencies exist
				for _, dep := range deps {
					if !slices.Contains(actualDeps, dep) {
						t.Errorf("missing dependency %s for release %s", dep, release)
					}
				}
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
			graph, err := NewDependencyGraph(test.releases)
			t.CheckError(false, err)
			err = graph.HasCycles()

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
				{Name: "a", DependsOn: []string{"b", "c"}},
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
			for level, releases := range test.expected {
				t.CheckDeepEqual(releases, levels[level])
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

func TestOrderPreservationWithinLevels(t *testing.T) {
	tests := []struct {
		description string
		releases    []latest.HelmRelease
		expected    map[int][]string
	}{
		{
			description: "preserve order within same level",
			releases: []latest.HelmRelease{
				{Name: "a3"},
				{Name: "a2"},
				{Name: "a1"},
				{Name: "b3", DependsOn: []string{"a3", "a2", "a1"}},
				{Name: "b2", DependsOn: []string{"a1", "a2", "a3"}},
				{Name: "b1", DependsOn: []string{"a1", "a2", "a3"}},
			},
			expected: map[int][]string{
				0: {"a3", "a2", "a1"},
				1: {"b3", "b2", "b1"},
			},
		},
		{
			description: "preserve order with mixed dependencies",
			releases: []latest.HelmRelease{
				{Name: "c2", DependsOn: []string{"a1"}},
				{Name: "a1"},
				{Name: "c1", DependsOn: []string{"a1"}},
				{Name: "c3", DependsOn: []string{"a1"}},
				{Name: "b1", DependsOn: []string{"c1", "c2", "c3"}},
			},
			expected: map[int][]string{
				0: {"a1"},
				1: {"c2", "c1", "c3"},
				2: {"b1"},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			graph, err := NewDependencyGraph(test.releases)
			t.CheckNoError(err)

			levels, err := graph.GetReleasesByLevel()
			t.CheckNoError(err)

			// Verify exact ordering within each level
			for level, expectedReleases := range test.expected {
				actualReleases, exists := levels[level]
				t.CheckTrue(exists)
				t.CheckDeepEqual(expectedReleases, actualReleases)
			}
		})
	}
}
