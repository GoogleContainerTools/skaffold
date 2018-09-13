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

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/karrick/godirwalk"
	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RetrieveImage is overridden for unit testing
var RetrieveImage = retrieveImage

func ValidateDockerfile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		logrus.Warnf("opening file %s: %s", path, err.Error())
		return false
	}
	res, err := parser.Parse(f)
	if err != nil || res == nil || len(res.AST.Children) == 0 {
		return false
	}
	// validate each node contains valid dockerfile directive
	for _, child := range res.AST.Children {
		_, ok := command.Commands[child.Value]
		if !ok {
			return false
		}
	}

	return true
}

func readDockerfile(workspace, absDockerfilePath string, buildArgs map[string]*string) ([]string, error) {
	f, err := os.Open(absDockerfilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "opening dockerfile: %s", absDockerfilePath)
	}
	defer f.Close()

	res, err := parser.Parse(f)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}

	var copied [][]string
	envs := map[string]string{}

	// First process all build args, and replace if necessary.
	for i, value := range res.AST.Children {
		switch value.Value {
		case command.Arg:
			var val, defaultValue, arg string
			argSetting := strings.Fields(value.Original)[1]

			argValues := strings.Split(argSetting, "=")

			if len(argValues) > 1 {
				arg, defaultValue = argValues[0], argValues[1]
			} else {
				arg = argValues[0]
			}

			valuePtr := buildArgs[arg]
			if valuePtr == nil && defaultValue == "" {
				logrus.Warnf("arg %s referenced in dockerfile but not provided with default or in build args", arg)
			} else {
				if valuePtr == nil {
					val = defaultValue
				} else {
					val = *valuePtr
				}
				if val == "" {
					logrus.Warnf("empty build arg provided in skaffold config: %s", arg)
					break
				}
				// we have a non-empty arg: replace it in all subsequent nodes
				for j := i; j < len(res.AST.Children); j++ {
					currentNode := res.AST.Children[j]
					for {
						if currentNode == nil {
							break
						}
						currentNode.Value = strings.Replace(currentNode.Value, "$"+arg, val, -1)
						currentNode.Value = strings.Replace(currentNode.Value, "${"+arg+"}", val, -1)
						currentNode = currentNode.Next
					}
				}
			}
		}
	}

	// Then process onbuilds, if present.
	onbuildsImages := [][]string{}
	stages := map[string]bool{}
	for _, value := range res.AST.Children {
		switch value.Value {
		case command.From:
			imageName := value.Next.Value
			if _, found := stages[imageName]; found {
				continue
			}

			next := value.Next.Next
			if next != nil && strings.ToLower(next.Value) == "as" {
				if next.Next != nil {
					stages[next.Next.Value] = true
				}
			}

			onbuilds, err := processBaseImage(imageName)
			if err != nil {
				logrus.Warnf("Error processing base image for onbuild triggers: %s. Dependencies may be incomplete.", err)
			}
			onbuildsImages = append(onbuildsImages, onbuilds)
		}
	}

	var dispatchInstructions = func(r *parser.Result) {
		for _, value := range r.AST.Children {
			switch value.Value {
			case command.Add, command.Copy:
				files, _ := processCopy(value, envs)
				if len(files) > 0 {
					copied = append(copied, files)
				}
			case command.Env:
				envs[value.Next.Value] = value.Next.Next.Value
			}
		}
	}
	for _, image := range onbuildsImages {
		for _, ob := range image {
			obRes, err := parser.Parse(strings.NewReader(ob))
			if err != nil {
				return nil, err
			}
			dispatchInstructions(obRes)
		}
	}

	dispatchInstructions(res)

	expandedPaths := make(map[string]bool)
	for _, files := range copied {
		matchesOne := false

		for _, p := range files {
			path := filepath.Join(workspace, p)
			if _, err := os.Stat(path); err == nil {
				expandedPaths[p] = true
				matchesOne = true
				continue
			}

			files, err := filepath.Glob(path)
			if err != nil {
				return nil, errors.Wrap(err, "invalid glob pattern")
			}
			if files == nil {
				continue
			}

			for _, f := range files {
				rel, err := filepath.Rel(workspace, f)
				if err != nil {
					return nil, fmt.Errorf("getting relative path of %s", f)
				}

				expandedPaths[rel] = true
			}
			matchesOne = true
		}

		if !matchesOne {
			return nil, fmt.Errorf("file pattern %s must match at least one file", files)
		}
	}

	var deps []string
	for dep := range expandedPaths {
		deps = append(deps, dep)
	}
	logrus.Infof("Found dependencies for dockerfile %s", deps)

	return deps, nil
}

// GetDependencies finds the sources dependencies for the given docker artifact.
// All paths are relative to the workspace.
func GetDependencies(workspace string, a *v1alpha3.DockerArtifact) ([]string, error) {
	absDockerfilePath, err := NormalizeDockerfilePath(workspace, a.DockerfilePath)
	if err != nil {
		return nil, errors.Wrap(err, "normalizing dockerfile path")
	}

	deps, err := readDockerfile(workspace, absDockerfilePath, a.BuildArgs)
	if err != nil {
		return nil, err
	}

	// Read patterns to ignore
	var excludes []string
	dockerignorePath := filepath.Join(workspace, ".dockerignore")
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
	}

	pExclude, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return nil, errors.Wrap(err, "invalid exclude patterns")
	}

	// Walk the workspace
	files := make(map[string]bool)
	for _, dep := range deps {
		dep = filepath.Clean(dep)
		absDep := filepath.Join(workspace, dep)

		fi, err := os.Stat(absDep)
		if err != nil {
			return nil, errors.Wrapf(err, "stating file %s", absDep)
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			if err := godirwalk.Walk(absDep, &godirwalk.Options{
				Unsorted: true,
				Callback: func(fpath string, info *godirwalk.Dirent) error {
					relPath, err := filepath.Rel(workspace, fpath)
					if err != nil {
						return err
					}

					ignored, err := pExclude.Matches(relPath)
					if err != nil {
						return err
					}

					if info.IsDir() {
						if ignored {
							return filepath.SkipDir
						}
					} else if !ignored {
						files[relPath] = true
					}

					return nil
				},
			}); err != nil {
				return nil, errors.Wrapf(err, "walking folder %s", absDep)
			}
		case mode.IsRegular():
			ignored, err := pExclude.Matches(dep)
			if err != nil {
				return nil, err
			}

			if !ignored {
				files[dep] = true
			}
		}
	}

	// Always add dockerfile even if it's .dockerignored. The daemon will need it anyways.
	if !filepath.IsAbs(a.DockerfilePath) {
		files[a.DockerfilePath] = true
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

	return dependencies, nil
}

func processBaseImage(baseImageName string) ([]string, error) {
	if strings.ToLower(baseImageName) == "scratch" {
		return nil, nil
	}

	logrus.Debugf("Checking base image %s for ONBUILD triggers.", baseImageName)
	img, err := RetrieveImage(baseImageName)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Found onbuild triggers %v in image %s", img.Config.OnBuild, baseImageName)
	return img.Config.OnBuild, nil
}

var imageCache sync.Map

func retrieveImage(image string) (*v1.ConfigFile, error) {
	cachedCfg, present := imageCache.Load(image)
	if present {
		return cachedCfg.(*v1.ConfigFile), nil
	}

	client, err := NewAPIClient()
	if err != nil {
		return nil, err
	}

	cfg := &v1.ConfigFile{}
	raw, err := retrieveLocalImage(client, image)
	if err == nil {
		if err := json.Unmarshal(raw, cfg); err != nil {
			return nil, err
		}
	} else {
		cfg, err = retrieveRemoteConfig(image)
		if err != nil {
			return nil, errors.Wrap(err, "getting remote config")
		}
	}

	imageCache.Store(image, cfg)

	return cfg, nil
}

func retrieveLocalImage(client APIClient, image string) ([]byte, error) {
	_, raw, err := client.ImageInspectWithRaw(context.Background(), image)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func retrieveRemoteConfig(identifier string) (*v1.ConfigFile, error) {
	img, err := remoteImage(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "getting image")
	}

	return img.ConfigFile()
}

func processCopy(value *parser.Node, envs map[string]string) ([]string, error) {
	var copied []string

	slex := shell.NewLex('\\')
	for {
		// Skip last node, since it is the destination, and stop if we arrive at a comment
		if value.Next.Next == nil || strings.HasPrefix(value.Next.Next.Value, "#") {
			break
		}
		src, err := processShellWord(slex, value.Next.Value, envs)
		if err != nil {
			return nil, errors.Wrap(err, "processing word")
		}
		// If the --from flag is provided, we are dealing with a multi-stage dockerfile
		// Adding a dependency from a different stage does not imply a source dependency
		if hasMultiStageFlag(value.Flags) {
			return nil, nil
		}
		if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
			copied = append(copied, src)
		} else {
			logrus.Debugf("Skipping watch on remote dependency %s", src)
		}

		value = value.Next
	}

	return copied, nil
}

func processShellWord(lex *shell.Lex, word string, envs map[string]string) (string, error) {
	envSlice := []string{}
	for envKey, envVal := range envs {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", envKey, envVal))
	}
	return lex.ProcessWord(word, envSlice)
}

func hasMultiStageFlag(flags []string) bool {
	for _, f := range flags {
		if strings.HasPrefix(f, "--from=") {
			return true
		}
	}
	return false
}
