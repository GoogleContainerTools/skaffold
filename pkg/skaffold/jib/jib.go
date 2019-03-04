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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// filesTemplate contains fields parsed from Jib's JSON output
type filesTemplate struct {
	Build  []string
	Inputs []string
	Ignore []string
}

// filesLists contains cached build/input dependencies
type filesLists struct {
	WatchedBuildFiles []string
	WatchedInputFiles []string
}

// watchedFiles maps from project name to watched files
var watchedFiles = map[string]filesLists{}

func getInputFiles(cmd *exec.Cmd, projectName string) ([]string, error) {
	if len(watchedFiles[projectName].WatchedInputFiles) == 0 && len(watchedFiles[projectName].WatchedBuildFiles) == 0 {
		// Refresh dependency list if empty
		if err := refreshDependencyList(cmd, projectName); err != nil {
			return nil, err
		}
	}
	return watchedFiles[projectName].WatchedInputFiles, nil
}

func getBuildFiles(cmd *exec.Cmd, projectName string) ([]string, error) {
	if len(watchedFiles[projectName].WatchedInputFiles) == 0 && len(watchedFiles[projectName].WatchedBuildFiles) == 0 {
		// Refresh dependency list if empty
		if err := refreshDependencyList(cmd, projectName); err != nil {
			return nil, err
		}
	}
	return watchedFiles[projectName].WatchedBuildFiles, nil
}

func refreshDependencyList(cmd *exec.Cmd, projectName string) error {
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return err
	}

	lines := util.NonEmptyLines(stdout)
	for i := range lines {
		if lines[i] == "BEGIN JIB JSON" {
			// Found Jib JSON header, next line is the JSON
			var filesOutput filesTemplate
			if err := json.Unmarshal([]byte(lines[i+1]), &filesOutput); err != nil {
				return err
			}

			// Walk the files in each list and filter out ignores
			files := watchedFiles[projectName]
			files.WatchedInputFiles, err = walkFiles(&filesOutput.Inputs, &filesOutput.Ignore)
			if err != nil {
				return err
			}
			files.WatchedBuildFiles, err = walkFiles(&filesOutput.Build, &filesOutput.Ignore)
			if err != nil {
				return err
			}
			watchedFiles[projectName] = files
			return nil
		}
	}

	return errors.New("failed to get Jib dependencies")
}

func walkFiles(filesOutputList *[]string, filesOutputIgnore *[]string) ([]string, error) {
	filesList := []string{}
	for _, dep := range *filesOutputList {
		if util.StrSliceContains(*filesOutputIgnore, dep) {
			continue
		}

		// Resolves directories recursively.
		info, err := os.Stat(dep)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Debugf("could not stat dependency: %s", err)
				continue // Ignore files that don't exist
			}
			return nil, errors.Wrapf(err, "unable to stat file %s", dep)
		}

		if !info.IsDir() {
			filesList = append(filesList, dep)
			continue
		}

		if err = godirwalk.Walk(dep, &godirwalk.Options{
			Unsorted: true,
			Callback: func(path string, _ *godirwalk.Dirent) error {
				if util.StrSliceContains(*filesOutputIgnore, path) {
					return filepath.SkipDir
				}
				filesList = append(filesList, path)
				return nil
			},
		}); err != nil {
			return nil, errors.Wrap(err, "filepath walk")
		}
	}
	return filesList, nil
}
