/*
Copyright 2020 The Skaffold Authors

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

package kustomize

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

// DependenciesForKustomization finds common kustomize artifacts relative to the
// provided working dir, and collects them into a list of files to be passed
// to the file watcher.
func DependenciesForKustomization(dir string) ([]string, error) {
	var deps []string

	path, err := FindKustomizationConfig(dir)
	if err != nil {
		// No kustomization config found so assume it's remote and stop traversing
		return deps, nil
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := kustomization{}
	if err := yaml.Unmarshal(buf, &content); err != nil {
		return nil, err
	}

	deps = append(deps, path)

	candidates := append(content.Bases, content.Resources...)
	candidates = append(candidates, content.Components...)

	for _, candidate := range candidates {
		// If the file doesn't exist locally, we can assume it's a remote file and
		// skip it, since we can't monitor remote files. Kustomize itself will
		// handle invalid/missing files.
		local, mode := pathExistsLocally(candidate, dir)
		if !local {
			continue
		}

		if mode.IsDir() {
			candidateDeps, err := DependenciesForKustomization(filepath.Join(dir, candidate))
			if err != nil {
				return nil, err
			}
			deps = append(deps, candidateDeps...)
		} else {
			deps = append(deps, filepath.Join(dir, candidate))
		}
	}

	for _, patch := range content.PatchesStrategicMerge {
		if patch.Path != "" {
			deps = append(deps, filepath.Join(dir, patch.Path))
		}
	}

	deps = append(deps, util.AbsolutePaths(dir, content.CRDs)...)

	for _, patch := range content.Patches {
		if patch.Path != "" {
			deps = append(deps, filepath.Join(dir, patch.Path))
		}
	}

	for _, jsonPatch := range content.PatchesJSON6902 {
		if jsonPatch.Path != "" {
			deps = append(deps, filepath.Join(dir, jsonPatch.Path))
		}
	}

	for _, generator := range content.ConfigMapGenerator {
		deps = append(deps, util.AbsolutePaths(dir, generator.Files)...)
		envs := generator.Envs
		if generator.Env != "" {
			envs = append(envs, generator.Env)
		}
		deps = append(deps, util.AbsolutePaths(dir, envs)...)
	}

	for _, generator := range content.SecretGenerator {
		deps = append(deps, util.AbsolutePaths(dir, generator.Files)...)
		envs := generator.Envs
		if generator.Env != "" {
			envs = append(envs, generator.Env)
		}
		deps = append(deps, util.AbsolutePaths(dir, envs)...)
	}

	return deps, nil
}

// FindKustomizationConfig finds the kustomization config relative to the provided dir.
// A Kustomization config must be at the root of the directory. Kustomize will
// error if more than one of these files exists so order doesn't matter.
func FindKustomizationConfig(dir string) (string, error) {
	for _, candidate := range kustomizeFilePaths {
		if local, _ := pathExistsLocally(candidate, dir); local {
			return filepath.Join(dir, candidate), nil
		}
	}
	return "", fmt.Errorf("no Kustomization configuration found in directory: %s", dir)
}

// BuildCommandArgs returns a list of build args to be passed to kustomize.
func BuildCommandArgs(buildArgs []string, kustomizePath string) []string {
	var args []string

	if len(buildArgs) > 0 {
		for _, v := range buildArgs {
			parts := strings.Split(v, " ")
			args = append(args, parts...)
		}
	}

	if len(kustomizePath) > 0 {
		args = append(args, kustomizePath)
	}

	return args
}
