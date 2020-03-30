/*
Copyright 2020 The Skaffold Authors

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

package list

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// Files list files in a workspace, given a list of patterns and exclusions.
// A pattern that doesn't correspond to any file is an error.
func Files(workspace string, patterns, excludes []string) ([]string, error) {
	var paths []string

	for _, pattern := range patterns {
		expanded, err := filepath.Glob(filepath.Join(workspace, pattern))
		if err != nil {
			return nil, err
		}

		if len(expanded) == 0 {
			return nil, fmt.Errorf("pattern %q did not match any file", pattern)
		}

		for _, e := range expanded {
			rel, err := filepath.Rel(workspace, e)
			if err != nil {
				return nil, err
			}

			paths = append(paths, rel)
		}
	}

	files, err := docker.WalkWorkspace(workspace, excludes, paths)
	if err != nil {
		return nil, fmt.Errorf("walking workspace %q: %w", workspace, err)
	}

	var dependencies []string
	for file := range files {
		dependencies = append(dependencies, file)
	}

	sort.Strings(dependencies)
	return dependencies, nil
}
