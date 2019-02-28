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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type filesTemplate struct {
	Build  []string
	Inputs []string
	Ignore []string
}

var watchedInputFiles, watchedBuildFiles []string

func getInputFiles(cmd *exec.Cmd) ([]string, error) {
	if len(watchedInputFiles) == 0 && len(watchedBuildFiles) == 0 {
		// Refresh dependency list if empty
		if err := refreshDependencyList(cmd); err != nil {
			return nil, err
		}
	}
	return watchedInputFiles, nil
}

func getBuildFiles(cmd *exec.Cmd) ([]string, error) {
	if len(watchedInputFiles) == 0 && len(watchedBuildFiles) == 0 {
		// Refresh dependency list if empty
		if err := refreshDependencyList(cmd); err != nil {
			return nil, err
		}
	}
	return watchedBuildFiles, nil
}

func refreshDependencyList(cmd *exec.Cmd) error {
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
			if err := walkFiles(&watchedBuildFiles, &filesOutput.Build, &filesOutput.Ignore); err != nil {
				return err
			}
			if err := walkFiles(&watchedInputFiles, &filesOutput.Inputs, &filesOutput.Ignore); err != nil {
				return err
			}

			return nil
		}
	}

	return errors.New("failed to get Jib dependencies")
}

func walkFiles(filesList *[]string, filesOutputList *[]string, filesOutputIgnore *[]string) error {
	*filesList = []string{}
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
			return errors.Wrapf(err, "unable to stat file %s", dep)
		}

		if !info.IsDir() {
			*filesList = append(*filesList, dep)
			continue
		}

		if err = godirwalk.Walk(dep, &godirwalk.Options{
			Unsorted: true,
			Callback: func(path string, _ *godirwalk.Dirent) error {
				if !util.StrSliceContains(*filesOutputIgnore, path) {
					*filesList = append(*filesList, path)
				}
				return nil
			},
		}); err != nil {
			return errors.Wrap(err, "filepath walk")
		}
	}
	return nil
}
