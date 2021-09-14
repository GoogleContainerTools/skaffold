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

	"github.com/bmatcuk/doublestar"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
)

// Files list files in a workspace, given a list of patterns and exclusions.
// It supports globstar (`**`) for recursive listing of subdirectories.
func Files(workspace string, patterns, excludes []string) ([]string, error) {
	// doublestar.Glob() doesn't find matches for patterns such as "."
	// Solve this by turning the relative workspace directory into an absolute path.
	workspace, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path of workspace %q: %w", workspace, err)
	}

	notExcluded := notExcluded(workspace, excludes)

	depSet := map[string]bool{}

	for _, pattern := range patterns {
		expanded, err := doublestar.Glob(filepath.Join(workspace, pattern))
		if err != nil {
			return nil, err
		}
		if len(expanded) == 0 {
			return nil, fmt.Errorf("pattern %q did not match any file", pattern)
		}

		for _, absFrom := range expanded {
			if err := walk.From(absFrom).Unsorted().When(notExcluded).WhenIsFile().Do(func(path string, info walk.Dirent) error {
				relPath, err := filepath.Rel(workspace, path)
				if err != nil {
					return err
				}
				depSet[relPath] = true
				return nil
			}); err != nil {
				return nil, fmt.Errorf("walking %q: %w", absFrom, err)
			}
		}
	}

	if len(depSet) == 0 {
		// Adding this conditional in order to be very (overly?) careful with backwards compatibility.
		// The previous implementation returned nil rather than empty slice in the case of no file dependencies.
		return nil, nil
	}

	dependencies := make([]string, len(depSet))
	i := 0
	for dep := range depSet {
		dependencies[i] = dep
		i++
	}
	sort.Strings(dependencies)
	return dependencies, nil
}

// notExcluded creates a walk.Predicate that matches file system entries
// only if they don't match a list of exclusion patterns.
func notExcluded(workspace string, excludes []string) walk.Predicate {
	return func(path string, info walk.Dirent) (bool, error) {
		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return false, err
		}

		for _, exclude := range excludes {
			matches, err := doublestar.PathMatch(exclude, relPath)
			if err != nil {
				return false, err
			}
			if matches {
				if info.IsDir() {
					return false, filepath.SkipDir
				}
				return false, nil
			}
		}

		return true, nil
	}
}
