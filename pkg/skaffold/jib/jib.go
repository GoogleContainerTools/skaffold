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
	"time"

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
	BuildFileTimes    map[string]time.Time
}

// watchedFiles maps from project name to watched files
var watchedFiles = map[string]filesLists{}

func getDependencies(cmd *exec.Cmd, projectName string) ([]string, error) {
	if len(watchedFiles[projectName].WatchedInputFiles) == 0 && len(watchedFiles[projectName].WatchedBuildFiles) == 0 {
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
		// Refresh dependency list if any build definitions have changed
		for _, buildFile := range watchedFiles[projectName].WatchedBuildFiles {
			info, err := os.Stat(buildFile)
			if err != nil {
				return nil, err
			}
			if val, ok := watchedFiles[projectName].BuildFileTimes[buildFile]; !ok || info.ModTime() != val {
				if err := refreshDependencyList(cmd, projectName); err != nil {
					return nil, err
				}
			}
		}
	}

	files := append(watchedFiles[projectName].WatchedBuildFiles, watchedFiles[projectName].WatchedInputFiles...)
	sort.Strings(files)
	return files, nil
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
			files.WatchedInputFiles, err = files.walkFiles(&filesOutput, false)
			if err != nil {
				return err
			}
			files.WatchedBuildFiles, err = files.walkFiles(&filesOutput, true)
			if err != nil {
				return err
			}
			watchedFiles[projectName] = files
			return nil
		}
	}

	return errors.New("failed to get Jib dependencies; it's possible you are using an old version of Jib (Skaffold requires Jib v1.0.2+)")
}

func (f *filesLists) walkFiles(jibOutput *filesTemplate, isBuildFile bool) ([]string, error) {
	filesList := []string{}
	filesOutputList := jibOutput.Inputs
	if isBuildFile {
		filesOutputList = jibOutput.Build
	}

	for _, dep := range filesOutputList {
		if util.StrSliceContains(jibOutput.Ignore, dep) {
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
			if isBuildFile {
				f.BuildFileTimes[dep] = info.ModTime()
			}
			continue
		}

		if err = godirwalk.Walk(dep, &godirwalk.Options{
			Unsorted: true,
			Callback: func(path string, _ *godirwalk.Dirent) error {
				if util.StrSliceContains(jibOutput.Ignore, path) {
					return filepath.SkipDir
				}
				filesList = append(filesList, path)
				if isBuildFile {
					f.BuildFileTimes[path] = info.ModTime()
				}
				return nil
			},
		}); err != nil {
			return nil, errors.Wrap(err, "filepath walk")
		}
	}
	return filesList, nil
}
