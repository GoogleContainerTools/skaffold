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
	"fmt"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// DependencyGraph represents a graph of helm release dependencies
type DependencyGraph struct {
	graph           map[string][]string
	releases        []latest.HelmRelease
	hasDependencies bool
}

// NewDependencyGraph creates a new DependencyGraph from a list of helm releases
func NewDependencyGraph(releases []latest.HelmRelease) (*DependencyGraph, error) {
	graph := make(map[string][]string)
	releaseNames := make(map[string]bool)

	for _, r := range releases {
		if _, exists := releaseNames[r.Name]; exists {
			return nil, fmt.Errorf("duplicate release name %s", r.Name)
		}
		releaseNames[r.Name] = true
	}

	// Check for non-existent dependencies
	hasDependencies := false
	for _, r := range releases {
		for _, dep := range r.DependsOn {
			if !releaseNames[dep] {
				return nil, fmt.Errorf("release %s depends on non-existent release %s", r.Name, dep)
			}
			hasDependencies = true
		}
		graph[r.Name] = r.DependsOn
	}

	g := &DependencyGraph{
		graph:           graph,
		releases:        releases,
		hasDependencies: hasDependencies,
	}

	if err := g.hasCycles(); err != nil {
		return nil, err
	}

	return g, nil
}

// GetReleasesByLevel returns releases grouped by their dependency level while preserving
// the original order within each level. Level 0 contains releases with no dependencies,
// level 1 contains releases that depend only on level 0 releases, and so on.
func (g *DependencyGraph) GetReleasesByLevel() (map[int][]string, error) {
	if len(g.releases) == 0 {
		// For empty releases, return empty map to avoid nil
		return map[int][]string{}, nil
	}

	if !g.hasDependencies {
		// Fast path:  if no dependencies, all releases are at level 0
		// Preserve original order from releases slice
		return map[int][]string{
			0: g.getNames(),
		}, nil
	}

	order, err := g.calculateDeploymentOrder()
	if err != nil {
		return nil, err
	}

	return g.groupReleasesByLevel(order), nil
}

// hasCycles checks if there are any cycles in the dependency graph
func (g *DependencyGraph) hasCycles() error {
	if !g.hasDependencies {
		return nil
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var checkCycle func(node string) error
	checkCycle = func(node string) error {
		if !visited[node] {
			visited[node] = true
			recStack[node] = true

			for _, dep := range g.graph[node] {
				if !visited[dep] {
					if err := checkCycle(dep); err != nil {
						return err
					}
				} else if recStack[dep] {
					return fmt.Errorf("cycle detected involving release %q", node)
				}
			}
		}
		recStack[node] = false
		return nil
	}

	for node := range g.graph {
		if !visited[node] {
			if err := checkCycle(node); err != nil {
				return err
			}
		}
	}
	return nil
}

// getNames returns a slice of release names in their original order
func (g *DependencyGraph) getNames() []string {
	names := make([]string, len(g.releases))
	for i, release := range g.releases {
		names[i] = release.Name
	}
	return names
}

// calculateDeploymentOrder returns a topologically sorted list of releases,
// ensuring that releases are deployed after their dependencies while maintaining
// the original order where possible
func (g *DependencyGraph) calculateDeploymentOrder() ([]string, error) {
	visited := make(map[string]bool)
	order := make([]string, 0, len(g.releases))

	// Create a mapping of release name to its index in original order
	originalOrder := make(map[string]int, len(g.releases))
	for i, release := range g.releases {
		originalOrder[release.Name] = i
	}

	var visit func(node string) error
	visit = func(node string) error {
		if visited[node] {
			return nil
		}
		visited[node] = true

		// Sort dependencies based on original order
		deps := make([]string, len(g.graph[node]))
		copy(deps, g.graph[node])
		if len(deps) > 1 {
			// Sort dependencies by their original position
			for i := 0; i < len(deps)-1; i++ {
				for j := i + 1; j < len(deps); j++ {
					if originalOrder[deps[i]] > originalOrder[deps[j]] {
						deps[i], deps[j] = deps[j], deps[i]
					}
				}
			}
		}

		// Visit dependencies in original order
		for _, dep := range deps {
			if err := visit(dep); err != nil {
				return err
			}
		}
		order = append(order, node)
		return nil
	}

	// Process releases in their original order
	for _, release := range g.releases {
		if err := visit(release.Name); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// groupReleasesByLevel groups releases by their dependency level while preserving
// the original order within each level
func (g *DependencyGraph) groupReleasesByLevel(order []string) map[int][]string {
	levels := make(map[int][]string)
	releaseLevels := make(map[string]int)

	// Calculate level for each release
	for _, release := range order {
		level := 0
		for _, dep := range g.graph[release] {
			if depLevel, exists := releaseLevels[dep]; exists {
				if depLevel >= level {
					level = depLevel + 1
				}
			}
		}
		releaseLevels[release] = level
		if levels[level] == nil {
			levels[level] = make([]string, 0)
		}
		levels[level] = append(levels[level], release)
	}

	return levels
}
