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
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	dotDotSlash = ".." + string(filepath.Separator)
)

// PluginType is an enum for the different supported Jib plugins.
type PluginType int

// Define the different plugin types supported by Jib.
const (
	JibMaven PluginType = iota
	JibGradle
)

// ID returns the identifier for a Jib plugin type, suitable for external references (YAML, JSON, command-line, etc).
func (t PluginType) ID() string {
	switch t {
	case JibMaven:
		return "maven"
	case JibGradle:
		return "gradle"
	}
	panic("Unknown Jib Plugin Type: " + string(t))
}

// Name provides a human-oriented label for a plugin type.
func (t PluginType) Name() string {
	switch t {
	case JibMaven:
		return "Jib Maven Plugin"
	case JibGradle:
		return "Jib Gradle Plugin"
	}
	panic("Unknown Jib Plugin Type: " + string(t))
}

// filesLists contains cached build/input dependencies
type filesLists struct {
	// BuildDefinitions lists paths to build definitions that trigger a call out to Jib to refresh the pathMap, as well as a rebuild, upon changing
	BuildDefinitions []string `json:"build"`

	// Inputs lists paths to build dependencies that trigger a rebuild upon changing
	Inputs []string `json:"inputs"`

	// Results lists paths to files that should be ignored when checking for changes to rebuild
	Results []string `json:"ignore"`

	// BuildFileTimes keeps track of the last modification time of each build file
	BuildFileTimes map[string]time.Time
}

// watchedFiles maps from project name to watched files
var watchedFiles = map[string]filesLists{}

// getDependencies returns a list of files to watch for changes to rebuild
func getDependencies(workspace string, cmd exec.Cmd, projectName string) ([]string, error) {
	var dependencyList []string
	files, ok := watchedFiles[projectName]
	if !ok {
		files = filesLists{}
	}

	if len(files.Inputs) == 0 && len(files.BuildDefinitions) == 0 {
		// Make sure build file modification time map is setup
		if files.BuildFileTimes == nil {
			files.BuildFileTimes = make(map[string]time.Time)
		}

		// Refresh dependency list if empty
		if err := refreshDependencyList(&files, cmd); err != nil {
			return nil, errors.Wrap(err, "initial Jib dependency refresh failed")
		}
	} else if err := walkFiles(workspace, files.BuildDefinitions, files.Results, func(path string, info os.FileInfo) error {
		// Walk build files to check for changes
		if val, ok := files.BuildFileTimes[path]; !ok || info.ModTime() != val {
			return refreshDependencyList(&files, cmd)
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to walk Jib build files for changes")
	}

	// Walk updated files to build dependency list
	if err := walkFiles(workspace, files.Inputs, files.Results, func(path string, info os.FileInfo) error {
		dependencyList = append(dependencyList, path)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to walk Jib input files to build dependency list")
	}
	if err := walkFiles(workspace, files.BuildDefinitions, files.Results, func(path string, info os.FileInfo) error {
		dependencyList = append(dependencyList, path)
		files.BuildFileTimes[path] = info.ModTime()
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to walk Jib build files to build dependency list")
	}

	// Store updated files list information
	watchedFiles[projectName] = files

	sort.Strings(dependencyList)
	return dependencyList, nil
}

// refreshDependencyList calls out to Jib to update files with the latest list of files/directories to watch.
func refreshDependencyList(files *filesLists, cmd exec.Cmd) error {
	stdout, err := util.RunCmdOut(&cmd)
	if err != nil {
		return errors.Wrap(err, "failed to get Jib dependencies")
	}

	// Search for Jib's output JSON. Jib's Maven/Gradle output takes the following form:
	// ...
	// BEGIN JIB JSON
	// {"build":["/paths","/to","/buildFiles"],"inputs":["/paths","/to","/inputs"],"ignore":["/paths","/to","/ignore"]}
	// ...
	// To parse the output, search for "BEGIN JIB JSON", then unmarshal the next line into the pathMap struct.
	matches := regexp.MustCompile(`BEGIN JIB JSON\r?\n({.*})`).FindSubmatch(stdout)
	if len(matches) == 0 {
		return errors.New("failed to get Jib dependencies")
	}

	line := bytes.Replace(matches[1], []byte(`\`), []byte(`\\`), -1)
	return json.Unmarshal(line, &files)
}

// walkFiles walks through a list of files and directories and performs a callback on each of the files
func walkFiles(workspace string, watchedFiles []string, ignoredFiles []string, callback func(path string, info os.FileInfo) error) error {
	// Skaffold prefers to deal with relative paths. In *practice*, Jib's dependencies
	// are *usually* absolute (relative to the root) and canonical (with all symlinks expanded).
	// But that's not guaranteed, so we try to relativize paths against the workspace as
	// both an absolute path and as a canonicalized workspace.
	workspaceRoots, err := calculateRoots(workspace)
	if err != nil {
		return errors.Wrapf(err, "unable to resolve workspace %s", workspace)
	}

	for _, dep := range watchedFiles {
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
			// try to relativize the path: an error indicates that the file cannot
			// be made relative to the roots, and so we just use the full path
			if relative, err := relativize(dep, workspaceRoots...); err == nil {
				dep = relative
			}
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
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					// try to relativize the path: an error indicates that the file cannot
					// be made relative to the roots, and so we just use the full path
					if relative, err := relativize(path, workspaceRoots...); err == nil {
						path = relative
					}
					return callback(path, info)
				}
				return nil
			},
		}); err != nil {
			return errors.Wrap(err, "filepath walk")
		}
	}
	return nil
}

// isIgnored tests a path for whether or not it should be ignored according to a list of ignored files/directories
func isIgnored(path string, ignoredFiles []string) bool {
	for _, ignored := range ignoredFiles {
		if strings.HasPrefix(path, ignored) {
			return true
		}
	}
	return false
}

// calculateRoots returns a list of possible symlink-expanded paths
func calculateRoots(path string) ([]string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to resolve %s", path)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to canonicalize workspace %s", path)
	}
	if path == canonical {
		return []string{path}, nil
	}
	return []string{canonical, path}, nil
}

// relativize tries to make path relative to one of the given roots
func relativize(path string, roots ...string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}
	for _, root := range roots {
		// check that the path can be made relative and is contained (since `filepath.Rel("/a", "/b") => "../b"`)
		if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, dotDotSlash) {
			return rel, nil
		}
	}
	return "", errors.New("could not relativize path")
}

// isOnInsecureRegistry checks if the given image specifies an insecure registry
func isOnInsecureRegistry(image string, insecureRegistries map[string]bool) (bool, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return false, err
	}

	registry := ref.Context().Registry.Name()
	return docker.IsInsecure(registry, insecureRegistries), nil
}
