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
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// filesTemplate contains fields parsed from Jib's JSON output
type filesTemplate struct {
	// Build lists paths to build definitions that trigger a call out to Jib to refresh the filesTemplate, as well as a rebuild, upon changing
	Build []string

	// Inputs lists paths to build dependencies that trigger a rebuild upon changing
	Inputs []string

	// Ignore lists paths to files that should be ignored when checking for changes to rebuild
	Ignore []string
}

// filesLists contains cached build/input dependencies
type filesLists struct {
	// FilesTemplate saves the most recent output of the Jib Skaffold files task/goal
	FilesTemplate filesTemplate

	// BuildFileTimes keeps track of the last modification time of each build file
	BuildFileTimes map[string]time.Time
}

// watchedFiles maps from project name to watched files
var watchedFiles = map[string]filesLists{}

// getDependencies returns a list of files to watch for changes to rebuild
func getDependencies(cmd *exec.Cmd, projectName string) ([]string, error) {
	dependencyList := []string{}
	template := watchedFiles[projectName].FilesTemplate
	if len(template.Inputs) == 0 && len(template.Build) == 0 {
		// Make sure build file modification time map is setup
		if watchedFiles[projectName].BuildFileTimes == nil {
			watched := watchedFiles[projectName]
			watched.BuildFileTimes = make(map[string]time.Time)
			watchedFiles[projectName] = watched
		}

		// Refresh dependency list if empty
		if err := refreshDependencyList(cmd, projectName); err != nil {
			return nil, err
		}
	} else {
		// Walk build files to check for changes
		if err := walkFiles(&template.Build, &template.Ignore, func(path string, info os.FileInfo) error {
			if val, ok := watchedFiles[projectName].BuildFileTimes[path]; !ok || info.ModTime() != val {
				return refreshDependencyList(cmd, projectName)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	// Walk updated files to build dependency list
	watched := watchedFiles[projectName]
	if err := walkFiles(&watched.FilesTemplate.Inputs, &watched.FilesTemplate.Ignore, func(path string, info os.FileInfo) error {
		dependencyList = append(dependencyList, path)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := walkFiles(&watched.FilesTemplate.Build, &watched.FilesTemplate.Ignore, func(path string, info os.FileInfo) error {
		dependencyList = append(dependencyList, path)
		watched.BuildFileTimes[path] = info.ModTime()
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(dependencyList)
	return dependencyList, nil
}

// refreshDependencyList calls out to Jib to retrieve an up-to-date list of files/directories to watch
func refreshDependencyList(cmd *exec.Cmd, projectName string) error {
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to get Jib dependencies; it's possible you are using an old version of Jib (Skaffold requires Jib v1.0.2+)")
	}

	lines := util.NonEmptyLines(stdout)
	for i := range lines {
		if lines[i] == "BEGIN JIB JSON" {
			// Found Jib JSON header, next line is the JSON
			files := watchedFiles[projectName]
			line := strings.Replace(lines[i+1], "\\", "\\\\", -1)
			if err := json.Unmarshal([]byte(line), &files.FilesTemplate); err != nil {
				return err
			}
			watchedFiles[projectName] = files
			return nil
		}
	}

	return errors.New("failed to get Jib dependencies")
}

// walkFiles walks through a list of files and directories and performs a callback on each of the files
func walkFiles(watchedFiles *[]string, ignoredFiles *[]string, callback func(path string, info os.FileInfo) error) error {
	for _, dep := range *watchedFiles {
		if isIgnored(dep, ignoredFiles) {
			continue
		}

		// Resolves directories recursively.
		info, err := os.Stat(dep)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Debugf("could not stat dependency: %s", err)
				continue // Ignore files that don't exist
			}
			return errors.Wrapf(err, "unable to stat file %s", dep)
		}

		// Process file
		if !info.IsDir() {
			if err := callback(dep, info); err != nil {
				return err
			}
			continue
		}

		// Process directory
		if err = godirwalk.Walk(dep, &godirwalk.Options{
			Unsorted: true,
			Callback: func(path string, _ *godirwalk.Dirent) error {
				if isIgnored(path, ignoredFiles) {
					return filepath.SkipDir
				}
				return callback(path, info)
			},
		}); err != nil {
			return errors.Wrap(err, "filepath walk")
		}
	}
	return nil
}

// isIgnored tests a path for whether or not it should be ignored according to a list of ignored files/directories
func isIgnored(path string, ignoredFiles *[]string) bool {
	for _, ignored := range *ignoredFiles {
		if strings.HasPrefix(path, ignored) {
			return true
		}
	}
	return false
}
