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

	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/moby/builder/dockerfile/parser"
	"github.com/moby/moby/builder/dockerfile/shell"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	add  = "add"
	copy = "copy"
	env  = "env"
	from = "from"
)

// RetrieveImage is overriden for unit testing
var RetrieveImage = retrieveImage

func readDockerfile(workspace, dockerfilePath string) ([]string, error) {
	path := filepath.Join(workspace, dockerfilePath)
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening dockerfile: %s", path)
	}
	defer f.Close()

	res, err := parser.Parse(f)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}

	envs := map[string]string{}
	depMap := map[string]struct{}{}
	// First process onbuilds, if present.
	onbuildsImages := [][]string{}
	for _, value := range res.AST.Children {
		switch value.Value {
		case from:
			onbuilds, err := processBaseImage(value)
			if err != nil {
				logrus.Warnf("Error processing base image for onbuild triggers: %s. Dependencies may be incomplete.", err)
			}
			onbuildsImages = append(onbuildsImages, onbuilds)
		}
	}

	var dispatchInstructions = func(r *parser.Result) {
		for _, value := range r.AST.Children {
			switch value.Value {
			case add, copy:
				processCopy(value, depMap, envs)
			case env:
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

	var deps []string
	for dep := range depMap {
		deps = append(deps, dep)
	}
	logrus.Infof("Found dependencies for dockerfile %s", deps)

	return deps, nil
}

func GetDependencies(dockerfilePath, workspace string) ([]string, error) {
	deps, err := readDockerfile(workspace, dockerfilePath)
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

	// Walk the workspace
	files := make(map[string]bool)
	for _, dep := range deps {
		filepath.Walk(filepath.Join(workspace, dep), func(fpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(workspace, fpath)
			if err != nil {
				return err
			}

			ignored, err := fileutils.Matches(relPath, excludes)
			if err != nil {
				return err
			}

			if info.IsDir() && ignored {
				return filepath.SkipDir
			}

			if !info.IsDir() && !ignored {
				files[relPath] = true
			}

			return nil
		})
	}

	// Always add dockerfile even if it's .dockerignored. The daemon will need it anyways.
	files[dockerfilePath] = true

	// Ignore .dockerignore
	delete(files, ".dockerignore")

	var dependencies []string
	for file := range files {
		dependencies = append(dependencies, file)
	}
	sort.Strings(dependencies)

	return dependencies, nil
}

func processBaseImage(value *parser.Node) ([]string, error) {
	base := value.Next.Value
	logrus.Debugf("Checking base image %s for ONBUILD triggers.", base)
	if strings.ToLower(base) == "scratch" {
		logrus.Debugf("SCRATCH base image found, skipping check: %s", base)
		return nil, nil
	}
	img, err := RetrieveImage(base)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Found onbuild triggers %v in image %s", img.Config.OnBuild, base)
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

func processCopy(value *parser.Node, paths map[string]struct{}, envs map[string]string) error {
	slex := shell.NewLex('\\')
	for {
		// Skip last node, since it is the destination, and stop if we arrive at a comment
		if value.Next.Next == nil || strings.HasPrefix(value.Next.Next.Value, "#") {
			break
		}
		src, err := processShellWord(slex, value.Next.Value, envs)
		if err != nil {
			return errors.Wrap(err, "processing word")
		}
		// If the --from flag is provided, we are dealing with a multi-stage dockerfile
		// Adding a dependency from a different stage does not imply a source dependency
		if hasMultiStageFlag(value.Flags) {
			return nil
		}
		if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
			paths[src] = struct{}{}
		} else {
			logrus.Debugf("Skipping watch on remote dependency %s", src)
		}

		value = value.Next
	}
	return nil
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
