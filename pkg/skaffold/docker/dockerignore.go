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

	"github.com/moby/patternmatcher"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/walk"
)

// NewDockerIgnorePredicate creates a walk.Predicate that checks if directory entries
// should be ignored.
func NewDockerIgnorePredicate(workspace string, excludes []string) (walk.Predicate, error) {
	matcher, err := patternmatcher.New(excludes)
	if err != nil {
		return nil, fmt.Errorf("invalid exclude patterns: %w", err)
	}

	return func(path string, info walk.Dirent) (bool, error) {
		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return false, err
		}
		ignored, err := matcher.MatchesOrParentMatches(relPath)
		if err != nil {
			return false, err
		}
		return ignored, nil
	}, nil
}
