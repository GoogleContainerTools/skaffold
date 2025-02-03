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

func TestBuildDependencyGraph(t *testing.T) {
	tests := []struct {
		description string
		releases    []latest.HelmRelease
		expected    map[string][]string
		shouldErr   bool
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
			description: "simple dependency graph with placeholder in name",
			releases: []latest.HelmRelease{
				{Name: "release1-{{.Service}}", DependsOn: []string{"release2-{{.Service}}"}},
				{Name: "release2-{{.Service}}", DependsOn: []string{}},
			},
			expected: map[string][]string{
				"release1-{{.Service}}": {"release2-{{.Service}}"},
				"release2-{{.Service}}": {},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			graph, err := BuildDependencyGraph(test.releases)

			if test.shouldErr {
				t.CheckError(true, err)
				return
			}

			t.CheckError(false, err)
			t.CheckDeepEqual(len(test.expected), len(graph))

			for release, deps := range test.expected {
				actualDeps, exists := graph[release]
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

func TestVerifyNoCycles(t *testing.T) {
	tests := []struct {
		description string
		graph       map[string][]string
		shouldErr   bool
	}{
		{
			description: "no cycles",
			graph: map[string][]string{
				"a": {"b", "c"},
				"b": {"c"},
				"c": {},
			},
			shouldErr: false,
		},
		{
			description: "simple cycle",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"a"},
			},
			shouldErr: true,
		},
		{
			description: "complex cycle",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"c"},
				"c": {"d"},
				"d": {"b"},
			},
			shouldErr: true,
		},
		{
			description: "self dependency",
			graph: map[string][]string{
				"a": {"a"},
			},
			shouldErr: true,
		},
		{
			description: "empty graph",
			graph:       map[string][]string{},
			shouldErr:   false,
		},
		{
			description: "disconnected components",
			graph: map[string][]string{
				"a": {"b"},
				"c": {"d"},
				"b": {},
				"d": {},
			},
			shouldErr: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := VerifyNoCycles(test.graph)

			if test.shouldErr {
				t.CheckErrorContains("cycle detected", err)
			} else {
				t.RequireNoError(err)
			}
		})
	}
}

func TestCalculateDeploymentOrder(t *testing.T) {
	tests := []struct {
		description string
		graph       map[string][]string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "linear dependency chain",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"c"},
				"c": {},
			},
			expected: []string{"c", "b", "a"},
		},
		{
			description: "multiple dependencies",
			graph: map[string][]string{
				"a": {"b", "c"},
				"b": {"d"},
				"c": {"d"},
				"d": {},
			},
			expected: []string{"d", "b", "c", "a"},
		},
		{
			description: "no dependencies",
			graph: map[string][]string{
				"a": {},
				"b": {},
				"c": {},
			},
			expected: []string{"a", "b", "c"},
		},
		{
			description: "diamond dependency",
			graph: map[string][]string{
				"a": {"b", "c"},
				"b": {"d"},
				"c": {"d"},
				"d": {},
			},
			expected: []string{"d", "b", "c", "a"},
		},
		{
			description: "empty graph",
			graph:       map[string][]string{},
			expected:    []string{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			order, err := calculateDeploymentOrder(test.graph)

			if test.shouldErr {
				t.CheckError(true, err)
				return
			}

			t.CheckError(false, err)

			// Verify order satisfies dependencies
			installed := make(map[string]bool)
			for _, release := range order {
				// Check all dependencies are installed
				for _, dep := range test.graph[release] {
					if !installed[dep] {
						t.Errorf("dependency %s not installed before %s", dep, release)
					}
				}
				installed[release] = true
			}

			// Verify all nodes are present
			if len(order) != len(test.graph) {
				t.Errorf("expected %d nodes, got %d", len(test.graph), len(order))
			}
		})
	}
}

func TestGroupReleasesByLevel(t *testing.T) {
	tests := []struct {
		description string
		order       []string
		graph       map[string][]string
		expected    map[int][]string
	}{
		{
			description: "linear chain",
			order:       []string{"c", "b", "a"},
			graph: map[string][]string{
				"a": {"b"},
				"b": {"c"},
				"c": {},
			},
			expected: map[int][]string{
				0: {"c"},
				1: {"b"},
				2: {"a"},
			},
		},
		{
			description: "multiple dependencies at same level",
			order:       []string{"d", "b", "c", "a"},
			graph: map[string][]string{
				"a": {"b", "c"},
				"b": {"d"},
				"c": {"d"},
				"d": {},
			},
			expected: map[int][]string{
				0: {"d"},
				1: {"b", "c"},
				2: {"a"},
			},
		},
		{
			description: "no dependencies",
			order:       []string{"a", "b", "c"},
			graph: map[string][]string{
				"a": {},
				"b": {},
				"c": {},
			},
			expected: map[int][]string{
				0: {"a", "b", "c"},
			},
		},
		{
			description: "empty graph",
			order:       []string{},
			graph:       map[string][]string{},
			expected:    map[int][]string{},
		},
		{
			description: "mixed levels",
			order:       []string{"d", "e", "b", "c", "a"},
			graph: map[string][]string{
				"a": {"b", "c"},
				"b": {"d"},
				"c": {"d", "e"},
				"d": {},
				"e": {},
			},
			expected: map[int][]string{
				0: {"d", "e"},
				1: {"b", "c"},
				2: {"a"},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			levels := groupReleasesByLevel(test.order, test.graph)

			t.CheckDeepEqual(len(test.expected), len(levels))

			for level, releases := range test.expected {
				t.CheckDeepEqual(releases, levels[level])
			}

			// Verify level assignments are correct
			for level, releases := range levels {
				for _, release := range releases {
					// Check that all dependencies are at lower levels
					for _, dep := range test.graph[release] {
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
		})
	}
}
