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

package docker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/fileutils"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
)

// NewDockerIgnorePredicate creates a walk.Predicate that checks if directory entries
// should be ignored.
func NewDockerIgnorePredicate(workspace string, excludes []string) (walk.Predicate, error) {
	matcher, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return nil, fmt.Errorf("invalid exclude patterns: %w", err)
	}

	return func(path string, info walk.Dirent) (bool, error) {
		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return false, err
		}

		ignored, err := matcher.Matches(relPath)
		if err != nil {
			return false, err
		}

		if ignored && info.IsDir() && skipDir(relPath, matcher) {
			return false, filepath.SkipDir
		}

		return ignored, nil
	}, nil
}

// exclusion handling closely follows vendor/github.com/docker/docker/pkg/archive/archive.go
func skipDir(relPath string, matcher *fileutils.PatternMatcher) bool {
	// No exceptions (!...) in patterns so just skip dir
	if !matcher.Exclusions() {
		return true
	}

	dirSlash := relPath + string(filepath.Separator)

	for _, pat := range matcher.Patterns() {
		if !pat.Exclusion() {
			continue
		}
		if strings.HasPrefix(pat.String()+string(filepath.Separator), dirSlash) {
			// found a match - so can't skip this dir
			return false
		}
	}

	return true
}
