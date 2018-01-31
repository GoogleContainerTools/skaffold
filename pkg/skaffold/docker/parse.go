/*
Copyright 2018 Google LLC

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
	"io"
	"os"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/third_party/moby/moby/dockerfile"
	"github.com/moby/moby/builder/dockerfile/parser"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const (
	add  = "add"
	copy = "copy"
	env  = "env"
)

var fs = afero.NewOsFs()

// GetDockerfileDependencies parses a dockerfile and returns the full paths
// of all the source files that the resulting docker image depends on.
func GetDockerfileDependencies(workspace string, r io.Reader) ([]string, error) {
	res, err := parser.Parse(r)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}
	envs := map[string]string{}
	depMap := map[string]struct{}{}
	for _, value := range res.AST.Children {
		switch value.Value {
		case add, copy:
			processCopy(workspace, value, depMap, envs)
		case env:
			envs[value.Next.Value] = value.Next.Next.Value
		}
	}

	deps := []string{}
	for dep := range depMap {
		deps = append(deps, dep)
	}

	expandedDeps, err := expandDeps(workspace, deps)
	if err != nil {
		return nil, errors.Wrap(err, "expanding dockerfile paths")
	}
	return expandedDeps, nil
}

func processCopy(workspace string, value *parser.Node, paths map[string]struct{}, envs map[string]string) error {
	slex := dockerfile.NewShellLex('\\')
	src, err := processShellWord(slex, value.Next.Value, envs)
	if err != nil {
		return errors.Wrap(err, "processing word")
	}
	// If the --from flag is provided, we are dealing with a multi-stage dockerfile
	// Adding a dependency from a different stage does not imply a source dependency
	if hasMultiStageFlag(value.Flags) {
		return nil
	}
	dep := path.Join(workspace, src)
	if _, ok := paths[dep]; ok {
		// If we've already seen this file, only add it once.
		return nil
	}
	paths[dep] = struct{}{}
	return nil
}

func expandDeps(workspace string, paths []string) ([]string, error) {
	expandedPaths := map[string]struct{}{}
	for _, p := range paths {
		// If the path contains a filepath.Match wildcard, we have to walk the workspace
		// and find any matches to that pattern
		if containsWildcards(p) {
			logrus.Debugf("COPY or ADD directive with wildcard %s", p)
			if err := afero.Walk(fs, workspace, func(fpath string, info os.FileInfo, err error) error {
				if err != nil {
					return errors.Wrap(err, "getting relative path")
				}
				if match, _ := path.Match(p, fpath); !match {
					return nil
				}
				if err := addFileOrDir(fs, fpath, info, expandedPaths); err != nil {
					return errors.Wrap(err, "adding file or directory")
				}
				return nil
			}); err != nil {
				return nil, errors.Wrap(err, "walking wildcard path")
			}
			continue
		}
		// If the path does not contain a wildcard, recursively add the directory or individual file
		info, err := fs.Stat(p)
		if err != nil {
			return nil, errors.Wrap(err, "getting file info")
		}
		if err := addFileOrDir(fs, p, info, expandedPaths); err != nil {
			return nil, errors.Wrap(err, "adding file or directory")
		}
	}
	ret := []string{}
	for ep := range expandedPaths {
		ret = append(ret, ep)
	}
	return ret, nil
}

func addFileOrDir(fs afero.Fs, ref string, info os.FileInfo, expandedPaths map[string]struct{}) error {
	if info.IsDir() {
		return addDir(fs, ref, expandedPaths)
	}
	expandedPaths[ref] = struct{}{}
	return nil
}

func addDir(fs afero.Fs, dir string, expandedPaths map[string]struct{}) error {
	logrus.Debugf("Recursively adding %s", dir)
	if err := afero.Walk(fs, dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		expandedPaths[path] = struct{}{}
		return nil
	}); err != nil {
		return errors.Wrap(err, "filepath walk")
	}
	return nil
}

func processShellWord(lex *dockerfile.ShellLex, word string, envs map[string]string) (string, error) {
	envSlice := []string{}
	for envKey, envVal := range envs {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", envKey, envVal))
	}
	return lex.ProcessWord(word, envSlice)
}

func containsWildcards(path string) bool {
	for i := 0; i < len(path); i++ {
		ch := path[i]
		// These are the wildcards that correspond to filepath.Match
		if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}

func hasMultiStageFlag(flags []string) bool {
	for _, f := range flags {
		if strings.HasPrefix(f, "--from=") {
			return true
		}
	}
	return false
}
