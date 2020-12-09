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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/docker/docker/builder/dockerignore"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
)

var (
	dependencyCache = util.NewSyncStore()
)

// BuildConfig encapsulates all the build configuration required for performing a dockerbuild.
type BuildConfig struct {
	workspace      string
	artifact       string
	dockerfilePath string
	args           map[string]*string
}

// NewBuildConfig returns a `BuildConfig` for a dockerfilePath build.
func NewBuildConfig(ws string, a string, path string, args map[string]*string) BuildConfig {
	return BuildConfig{
		workspace:      ws,
		artifact:       a,
		dockerfilePath: path,
		args:           args,
	}
}

// NormalizeDockerfilePath returns the absolute path to the dockerfilePath.
func NormalizeDockerfilePath(context, dockerfile string) (string, error) {
	// Expected case: should be found relative to the context directory.
	// If it does not exist, check if it's found relative to the current directory in case it's shared.
	// Otherwise return the path relative to the context directory, where it should have been.
	rel := filepath.Join(context, dockerfile)
	if _, err := os.Stat(rel); os.IsNotExist(err) {
		if _, err := os.Stat(dockerfile); err == nil || !os.IsNotExist(err) {
			return filepath.Abs(dockerfile)
		}
	}
	return filepath.Abs(rel)
}

// GetDependencies finds the sources dependency for the given docker artifact.
// it caches the results for the computed dependency which can be used by `GetDependenciesCached`
// All paths are relative to the workspace.
func GetDependencies(ctx context.Context, buildCfg BuildConfig, cfg Config) ([]string, error) {
	absDockerfilePath, err := NormalizeDockerfilePath(buildCfg.workspace, buildCfg.dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("normalizing dockerfilePath path: %w", err)
	}
	result := getDependencies(buildCfg.workspace, buildCfg.dockerfilePath, absDockerfilePath, buildCfg.args, cfg)
	dependencyCache.Store(buildCfg.artifact, result)
	return resultPair(result)
}

// GetDependenciesCached reads from cache finds the sources dependency for the given docker artifact.
// All paths are relative to the workspace.
func GetDependenciesCached(ctx context.Context, buildCfg BuildConfig, cfg Config) ([]string, error) {
	absDockerfilePath, err := NormalizeDockerfilePath(buildCfg.workspace, buildCfg.dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("normalizing dockerfilePath path: %w", err)
	}

	deps := dependencyCache.Exec(buildCfg.artifact, func() interface{} {
		return getDependencies(buildCfg.workspace, buildCfg.dockerfilePath, absDockerfilePath, buildCfg.args, cfg)
	})
	return resultPair(deps)
}

func resultPair(deps interface{}) ([]string, error) {
	switch t := deps.(type) {
	case error:
		return nil, t
	case []string:
		return t, nil
	default:
		return nil, fmt.Errorf("internal error when retrieving cache result of type %T", t)
	}
}

func getDependencies(workspace string, dockerfilePath string, absDockerfilePath string, buildArgs map[string]*string, cfg Config) interface{} {
	// If the Dockerfile doesn't exist, we can't compute the dependency.
	// But since we know the Dockerfile is a dependency, let's return a list
	// with only that file. It makes errors down the line more actionable
	// than returning an error now.
	if _, err := os.Stat(absDockerfilePath); os.IsNotExist(err) {
		return []string{dockerfilePath}
	}

	fts, err := readCopyCmdsFromDockerfile(false, absDockerfilePath, workspace, buildArgs, cfg)
	if err != nil {
		return err
	}

	excludes, err := readDockerignore(workspace, absDockerfilePath)
	if err != nil {
		return fmt.Errorf("reading .dockerignore: %w", err)
	}

	deps := make([]string, 0, len(fts))
	for _, ft := range fts {
		deps = append(deps, ft.from)
	}

	files, err := WalkWorkspace(workspace, excludes, deps)
	if err != nil {
		return fmt.Errorf("walking workspace: %w", err)
	}

	// Always add dockerfile even if it's .dockerignored. The daemon will need it anyways.
	if !filepath.IsAbs(dockerfilePath) {
		files[dockerfilePath] = true
	} else {
		files[absDockerfilePath] = true
	}

	// Ignore .dockerignore
	delete(files, ".dockerignore")

	var dependencies []string
	for file := range files {
		dependencies = append(dependencies, file)
	}
	sort.Strings(dependencies)

	return dependencies
}

// readDockerignore reads patterns to ignore
func readDockerignore(workspace string, absDockerfilePath string) ([]string, error) {
	var excludes []string
	dockerignorePaths := []string{
		absDockerfilePath + ".dockerignore",
		filepath.Join(workspace, ".dockerignore"),
	}
	for _, dockerignorePath := range dockerignorePaths {
		if _, err := os.Stat(dockerignorePath); !os.IsNotExist(err) {
			r, err := os.Open(dockerignorePath)
			if err != nil {
				return nil, err
			}
			defer r.Close()

			excludes, err = dockerignore.ReadAll(r)
			if err != nil {
				return nil, err
			}
			return excludes, nil
		}
	}
	return nil, nil
}

// WalkWorkspace walks the given host directories and records all files found.
// Note: if you change this function, you might also want to modify `walkWorkspaceWithDestinations`.
func WalkWorkspace(workspace string, excludes, deps []string) (map[string]bool, error) {
	dockerIgnored, err := NewDockerIgnorePredicate(workspace, excludes)
	if err != nil {
		return nil, err
	}

	// Walk the workspace
	files := make(map[string]bool)
	for _, dep := range deps {
		absFrom := filepath.Join(workspace, dep)

		keepFile := func(path string, info walk.Dirent) (bool, error) {
			// Always keep root folders.
			if info.IsDir() && path == absFrom {
				return true, nil
			}

			ignored, err := dockerIgnored(path, info)
			if err != nil {
				return false, err
			}

			return !ignored, nil
		}

		if err := walk.From(absFrom).Unsorted().When(keepFile).WhenIsFile().Do(func(path string, info walk.Dirent) error {
			relPath, err := filepath.Rel(workspace, path)
			if err != nil {
				return err
			}

			files[relPath] = true
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walking %q: %w", absFrom, err)
		}
	}

	return files, nil
}
