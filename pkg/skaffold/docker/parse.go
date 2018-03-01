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
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/containers/image/types"

	"github.com/containers/image/manifest"

	"github.com/containers/image/docker"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/moby/moby/builder/dockerfile/parser"
	"github.com/moby/moby/builder/dockerfile/shell"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	add    = "add"
	copy   = "copy"
	env    = "env"
	from   = "from"
	expose = "expose"
)

// For testing.
var RetrieveConfig = retrieveImageConfig

// GetDockerfileDependencies parses a dockerfile and returns the full paths
// of all the source files that the resulting docker image depends on.
func GetDockerfileDependencies(workspace string, r io.Reader) ([]string, error) {
	res, err := parser.Parse(r)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}
	envs := map[string]string{}
	depMap := map[string]struct{}{}
	// First process onbuilds, if present.
	onbuilds := []string{}
	for _, value := range res.AST.Children {
		switch value.Value {
		case from:
			onbuilds, err = processBaseImage(value)
			if err != nil {
				logrus.Warnf("Error processing base image for onbuild triggers: %s. Dependencies may be incomplete.", err)
			}
		}
	}

	var dispatchInstructions = func(r *parser.Result) {
		for _, value := range r.AST.Children {
			switch value.Value {
			case add, copy:
				processCopy(workspace, value, depMap, envs)
			case env:
				envs[value.Next.Value] = value.Next.Next.Value
			}
		}
	}
	for _, ob := range onbuilds {
		obRes, err := parser.Parse(strings.NewReader(ob))
		if err != nil {
			return nil, err
		}
		dispatchInstructions(obRes)
	}

	dispatchInstructions(res)

	deps := []string{}
	for dep := range depMap {
		deps = append(deps, dep)
	}
	logrus.Infof("Found dependencies for dockerfile %s", deps)
	expandedDeps, err := util.ExpandPaths(workspace, deps)
	if err != nil {
		return nil, errors.Wrap(err, "expanding dockerfile paths")
	}

	// Look for .dockerignore.
	ignorePath := filepath.Join(workspace, ".dockerignore")

	filteredDeps, err := util.ApplyDockerIgnore(expandedDeps, ignorePath)
	if err != nil {
		return nil, errors.Wrap(err, "applying dockerignore")
	}
	return filteredDeps, nil
}

func PortsFromDockerfile(r io.Reader) ([]string, error) {
	res, err := parser.Parse(r)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}

	// Check the dockerfile and the base.
	ports := []string{}
	for _, value := range res.AST.Children {
		switch value.Value {
		case from:
			base := value.Next.Value
			if base == "SCRATCH" {
				logrus.Debug("Skipping port check in SCRATCH base image.")
				continue
			}
			config, err := RetrieveConfig(value.Next.Value)
			if err != nil {
				logrus.Warnf("Error checking base image for ports: %s", err)
				continue
			}
			for port := range config.Config.ExposedPorts {
				logrus.Debugf("Found port %s in base image", port)
				ports = append(ports, string(port))
			}
		case expose:
			// There can be multiple ports per line.
			for {
				if value.Next == nil {
					break
				}
				port := value.Next.Value
				logrus.Debugf("Found port %s in Dockerfile", port)
				ports = append(ports, port)
				value = value.Next
			}
		}
	}
	return ports, nil
}

func processBaseImage(value *parser.Node) ([]string, error) {
	base := value.Next.Value
	logrus.Debugf("Checking base image %s for ONBUILD triggers.", base)
	if strings.ToLower(base) == "scratch" {
		logrus.Debugf("SCRATCH base image found, skipping check: %s", base)
		return nil, nil
	}
	cfg, err := RetrieveConfig(base)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Found onbuild triggers %v in image %s", cfg.Config.OnBuild, base)
	return cfg.Config.OnBuild, nil
}

func retrieveImageConfig(image string) (*manifest.Schema2Image, error) {
	ref, err := docker.ParseReference("//" + image)
	if err != nil {
		return nil, err
	}

	context := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "amd64",
	}
	img, err := ref.NewImage(context)

	if err != nil {
		return nil, err
	}

	cfgBytes, err := img.ConfigBlob()
	if err != nil {
		return nil, err
	}
	cfg := &manifest.Schema2Image{}
	if err := json.Unmarshal(cfgBytes, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func processCopy(workspace string, value *parser.Node, paths map[string]struct{}, envs map[string]string) error {
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
		dep := path.Join(workspace, src)
		paths[dep] = struct{}{}
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
