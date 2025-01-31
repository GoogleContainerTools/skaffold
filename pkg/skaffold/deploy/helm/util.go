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
	"fmt"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

func BuildDependencyGraph(releases []latest.HelmRelease) (map[string][]string, error) {
	dependencyGraph := make(map[string][]string)
	for _, r := range releases {
		dependencyGraph[r.Name] = r.DependsOn
	}

	return dependencyGraph, nil
}

// VerifyNoCycles checks if there are any cycles in the dependency graph
func VerifyNoCycles(graph map[string][]string) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var checkCycle func(node string) error
	checkCycle = func(node string) error {
		if !visited[node] {
			visited[node] = true
			recStack[node] = true

			for _, dep := range graph[node] {
				if !visited[dep] {
					if err := checkCycle(dep); err != nil {
						return err
					}
				} else if recStack[dep] {
					return fmt.Errorf("cycle detected involving release %s", node)
				}
			}
		}
		recStack[node] = false
		return nil
	}

	for node := range graph {
		if !visited[node] {
			if err := checkCycle(node); err != nil {
				return err
			}
		}
	}
	return nil
}

// calculateDeploymentOrder returns a topologically sorted list of releases,
// ensuring that releases are deployed after their dependencies.
func calculateDeploymentOrder(graph map[string][]string) ([]string, error) {
	visited := make(map[string]bool)
	order := make([]string, 0)

	var visit func(node string) error
	visit = func(node string) error {
		if visited[node] {
			return nil
		}
		visited[node] = true

		for _, dep := range graph[node] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		order = append(order, node)
		return nil
	}

	for node := range graph {
		if err := visit(node); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// groupReleasesByLevel groups releases by their dependency level
// Level 0 contains releases with no dependencies
// Level 1 contains releases that depend only on level 0 releases
// And so on...
func groupReleasesByLevel(order []string, graph map[string][]string) map[int][]string {
	levels := make(map[int][]string)
	releaseLevels := make(map[string]int)

	// Calculate level for each release
	for _, release := range order {
		level := 0
		for _, dep := range graph[release] {
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
