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

package jib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func getDependencies(workspace string, cmd *exec.Cmd) ([]string, error) {
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, err
	}

	// Jib's dependencies are absolute, and usually canonicalized, so must canonicalize the workspace
	if workspace, err = filepath.Abs(workspace); err != nil {
		return nil, errors.Wrapf(err, "unable to resolve workspace %s", workspace)
	}
	canonicalWorkspace, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to canonicalize workspace %s", workspace)
	}

	cmdDirInfo, err := os.Stat(cmd.Dir)
	if err != nil {
		return nil, err
	}

	// Parses stdout for the dependencies, one per line
	lines := util.NonEmptyLines(stdout)

	var deps []string
	for _, dep := range lines {
		// Resolves directories recursively.
		info, err := os.Stat(dep)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Debugf("could not stat dependency: %s", err)
				continue // Ignore files that don't exist
			}
			return nil, errors.Wrapf(err, "unable to stat file %s", dep)
		}

		// TODO(coollog): Remove this once Jib deps are prepended with special sequence.
		// Skips the project directory itself. This is necessary as some wrappers print the project directory for some reason.
		if os.SameFile(cmdDirInfo, info) {
			continue
		}

		if !info.IsDir() {
			// try to relativize the path
			if relative, err := relativize(dep, workspace, canonicalWorkspace); err == nil {
				dep = relative
			}
			deps = append(deps, dep)
			continue
		}

		if err = godirwalk.Walk(dep, &godirwalk.Options{
			Unsorted: true,
			Callback: func(path string, _ *godirwalk.Dirent) error {
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					// try to relativize the path
					if relative, err := relativize(path, workspace, canonicalWorkspace); err == nil {
						path = relative
					}
					deps = append(deps, path)
				}
				return nil
			},
		}); err != nil {
			return nil, errors.Wrap(err, "filepath walk")
		}
	}

	sort.Strings(deps)
	return deps, nil
}

// relativize tries to make path relative to the root location(s)
func relativize(path string, roots ...string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}
	dotDotSlash := fmt.Sprintf("..%c", filepath.Separator)
	for _, root := range roots {
		// check that the path can be made relative and is contained (since `filepath.Rel("/a", "/b") => "../b"`)
		if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, dotDotSlash) {
			return rel, nil
		}
	}
	return "", errors.New("could not relativize path")
}
