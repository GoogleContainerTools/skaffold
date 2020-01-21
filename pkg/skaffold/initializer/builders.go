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

package initializer

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type builderAnalyzer struct {
	directoryAnalyzer
	enableJibInit       bool
	enableBuildpackInit bool
	findBuilders        bool
	foundBuilders       []InitBuilder

	parentDirToStopFindBuilders string
}

func (a *builderAnalyzer) analyzeFile(filePath string) error {
	if a.findBuilders && (a.parentDirToStopFindBuilders == "" || a.parentDirToStopFindBuilders == a.currentDir) {
		builderConfigs, continueSearchingBuilders := detectBuilders(a.enableJibInit, a.enableBuildpackInit, filePath)
		a.foundBuilders = append(a.foundBuilders, builderConfigs...)
		if !continueSearchingBuilders {
			a.parentDirToStopFindBuilders = a.currentDir
		}
	}
	return nil
}

func (a *builderAnalyzer) exitDir(dir string) {
	if a.parentDirToStopFindBuilders == dir {
		a.parentDirToStopFindBuilders = ""
	}
}

// autoSelectBuilders takes a list of builders and images, checks if any of the builders' configured target
// images match an image in the image list, and returns a list of the matching builder/image pairs. Also
// separately returns the builder configs and images that didn't have any matches.
func autoSelectBuilders(builderConfigs []InitBuilder, images []string) ([]builderImagePair, []InitBuilder, []string) {
	var pairs []builderImagePair
	var unresolvedImages = make(sortedSet)
	for _, image := range images {
		matchingConfigIndex := -1
		for i, config := range builderConfigs {
			if image != config.ConfiguredImage() {
				continue
			}

			// Found more than one match; can't auto-select.
			if matchingConfigIndex != -1 {
				matchingConfigIndex = -1
				break
			}
			matchingConfigIndex = i
		}

		if matchingConfigIndex != -1 {
			// Exactly one pair found; save the pair and remove from remaining build configs
			pairs = append(pairs, builderImagePair{ImageName: image, Builder: builderConfigs[matchingConfigIndex]})
			builderConfigs = append(builderConfigs[:matchingConfigIndex], builderConfigs[matchingConfigIndex+1:]...)
		} else {
			// No definite pair found, add to images list
			unresolvedImages.add(image)
		}
	}
	return pairs, builderConfigs, unresolvedImages.values()
}

// detectBuilders checks if a path is a builder config, and if it is, returns the InitBuilders representing the
// configs. Also returns a boolean marking search completion for subdirectories (true = subdirectories should
// continue to be searched, false = subdirectories should not be searched for more builders)
func detectBuilders(enableJibInit, enableBuildpackInit bool, path string) ([]InitBuilder, bool) {
	// TODO: Remove backwards compatibility if statement (not entire block)
	if enableJibInit {
		// Check for jib
		if builders := jib.Validate(path); builders != nil {
			results := make([]InitBuilder, len(builders))
			for i := range builders {
				results[i] = builders[i]
			}
			return results, false
		}
	}

	// Check for Dockerfile
	base := filepath.Base(path)
	if strings.Contains(strings.ToLower(base), "dockerfile") {
		if docker.Validate(path) {
			results := []InitBuilder{docker.ArtifactConfig{File: path}}
			return results, true
		}
	}

	// TODO: Remove backwards compatibility if statement (not entire block)
	if enableBuildpackInit {
		// Check for buildpacks
		if buildpacks.Validate(path) {
			results := []InitBuilder{buildpacks.ArtifactConfig{File: path}}
			return results, true
		}
	}

	// TODO: Check for more builders

	return nil, true
}

func processCliArtifacts(artifacts []string) ([]builderImagePair, error) {
	var pairs []builderImagePair
	for _, artifact := range artifacts {
		// Parses JSON in the form of: {"builder":"Name of Builder","payload":{...},"image":"image.name"}.
		// The builder field is parsed first to determine the builder type, and the payload is parsed
		// afterwards once the type is determined.
		a := struct {
			Name  string `json:"builder"`
			Image string `json:"image"`
		}{}
		if err := json.Unmarshal([]byte(artifact), &a); err != nil {
			// Not JSON, use backwards compatible method
			parts := strings.Split(artifact, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("malformed artifact provided: %s", artifact)
			}
			pairs = append(pairs, builderImagePair{
				Builder:   docker.ArtifactConfig{File: parts[0]},
				ImageName: parts[1],
			})
			continue
		}

		// Use builder type to parse payload
		switch a.Name {
		case docker.Name:
			parsed := struct {
				Payload docker.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			pair := builderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		// FIXME: shouldn't use a human-readable name?
		case jib.PluginName(jib.JibGradle), jib.PluginName(jib.JibMaven):
			parsed := struct {
				Payload jib.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			parsed.Payload.BuilderName = a.Name
			pair := builderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		case buildpacks.Name:
			parsed := struct {
				Payload buildpacks.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			pair := builderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		default:
			return nil, errors.New("unknown builder type in CLI artifacts")
		}
	}
	return pairs, nil
}

// For each image parsed from all k8s manifests, prompt the user for the builder that builds the referenced image
func resolveBuilderImages(builderConfigs []InitBuilder, images []string, force bool) ([]builderImagePair, error) {
	// If nothing to choose, don't bother prompting
	if len(images) == 0 || len(builderConfigs) == 0 {
		return []builderImagePair{}, nil
	}

	// if we only have 1 image and 1 build config, don't bother prompting
	if len(images) == 1 && len(builderConfigs) == 1 {
		return []builderImagePair{{
			Builder:   builderConfigs[0],
			ImageName: images[0],
		}}, nil
	}

	if force {
		return nil, errors.New("unable to automatically resolve builder/image pairs; run `skaffold init` without `--force` to manually resolve ambiguities")
	}

	// Build map from choice string to builder config struct
	choices := make([]string, len(builderConfigs))
	choiceMap := make(map[string]InitBuilder, len(builderConfigs))
	for i, buildConfig := range builderConfigs {
		choice := buildConfig.Describe()
		choices[i] = choice
		choiceMap[choice] = buildConfig
	}
	sort.Strings(choices)

	// For each choice, use prompt string to pair builder config with k8s image
	pairs := []builderImagePair{}
	for {
		if len(images) == 0 {
			break
		}

		image := images[0]
		choice, err := promptUserForBuildConfigFunc(image, choices)
		if err != nil {
			return nil, err
		}

		if choice != NoBuilder {
			pairs = append(pairs, builderImagePair{Builder: choiceMap[choice], ImageName: image})
			choices = util.RemoveFromSlice(choices, choice)
		}
		images = util.RemoveFromSlice(images, image)
	}
	if len(choices) > 0 {
		logrus.Warnf("unused builder configs found in repository: %v", choices)
	}
	return pairs, nil
}
