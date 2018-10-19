/*
Copyright 2018 The Skaffold Authors

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
	"os/exec"
	"sort"

	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func getDependencies(cmd *exec.Cmd) ([]string, error) {
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, err
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
			deps = append(deps, dep)
			continue
		}

		if err = godirwalk.Walk(dep, &godirwalk.Options{
			Unsorted: true,
			Callback: func(path string, _ *godirwalk.Dirent) error {
				deps = append(deps, path)
				return nil
			},
		}); err != nil {
			return nil, errors.Wrap(err, "filepath walk")
		}
	}

	sort.Strings(deps)
	return deps, nil
}
