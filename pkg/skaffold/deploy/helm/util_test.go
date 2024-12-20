package helm

import (
	"slices"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	"github.com/stretchr/testify/require"
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
			description: "invalid release name template",
			releases: []latest.HelmRelease{
				{Name: "{{.Invalid}}"},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			graph, err := BuildDependencyGraph(test.releases)

			if test.shouldErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, graph)
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
				require.Error(t, err)
				require.Contains(t, err.Error(), "cycle detected")
			} else {
				require.NoError(t, err)
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
		t.Run(test.description, func(t *testing.T) {
			order, err := calculateDeploymentOrder(test.graph)

			if test.shouldErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(test.expected), len(order), "deployment order length mismatch")

			// Verify order satisfies dependencies
			installed := make(map[string]bool)
			for _, release := range order {
				// Check all dependencies are installed
				for _, dep := range test.graph[release] {
					require.True(t, installed[dep],
						"release %s deployed before dependency %s", release, dep)
				}
				installed[release] = true
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
		t.Run(test.description, func(t *testing.T) {
			levels := groupReleasesByLevel(test.order, test.graph)

			require.Equal(t, len(test.expected), len(levels), "number of levels mismatch")

			for level, releases := range test.expected {
				require.ElementsMatch(t, releases, levels[level],
					"releases at level %d don't match", level)
			}

			// Verify level assignments are correct
			for level, releases := range levels {
				for _, release := range releases {
					// Check that all dependencies are at lower levels
					for _, dep := range test.graph[release] {
						for depLevel, depReleases := range levels {
							if slices.Contains(depReleases, dep) {
								require.Less(t, depLevel, level,
									"dependency %s at level %d >= release %s at level %d",
									dep, depLevel, release, level)
							}
						}
					}
				}
			}
		})
	}
}
