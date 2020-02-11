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

package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

// NoBuilder allows users to specify they don't want to build
// an image we parse out from a Kubernetes manifest
const NoBuilder = "None (image not built from these sources)"

// InitBuilder represents a builder that can be chosen by skaffold init.
type InitBuilder interface {
	// Name returns the name of the builder
	Name() string
	// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
	// Must be unique between artifacts.
	Describe() string
	// UpdateArtifact updates the Artifact to be included in the generated Build Config
	UpdateArtifact(*latest.Artifact)
	// ConfiguredImage returns the target image configured by the builder, or an empty string if no image is configured.
	// This should be a cheap operation.
	ConfiguredImage() string
	// Path returns the path to the build file
	Path() string
}

// BuilderImagePair defines a builder and the image it builds
type BuilderImagePair struct {
	Builder   InitBuilder
	ImageName string
}

// MatchBuildersToImages takes a list of builders and images, checks if any of the builders' configured target
// images match an image in the image list, and returns a list of the matching builder/image pairs. Also
// separately returns the builder configs and images that didn't have any matches.
func MatchBuildersToImages(builderConfigs []InitBuilder, images []string) ([]BuilderImagePair, []InitBuilder, []string) {
	var pairs []BuilderImagePair
	var unresolvedImages = make(sortedSet)
	for _, image := range images {
		builderIdx := findExactlyOnceMatchingBuilder(builderConfigs, image)

		// exactly one builder found for the image
		if builderIdx != -1 {
			// save the pair
			pairs = append(pairs, BuilderImagePair{ImageName: image, Builder: builderConfigs[builderIdx]})
			// remove matched builder from builderConfigs
			builderConfigs = append(builderConfigs[:builderIdx], builderConfigs[builderIdx+1:]...)
		} else {
			// No definite pair found, add to images list
			unresolvedImages.add(image)
		}
	}
	return pairs, builderConfigs, unresolvedImages.values()
}

func findExactlyOnceMatchingBuilder(builderConfigs []InitBuilder, image string) int {
	matchingConfigIndex := -1
	for i, config := range builderConfigs {
		if image != config.ConfiguredImage() {
			continue
		}
		// Found more than one match;
		if matchingConfigIndex != -1 {
			return -1
		}
		matchingConfigIndex = i
	}
	return matchingConfigIndex
}

// TODO(nkubala): make these private again once DoInit() relinquishes control of the builder/image processing
func ProcessCliArtifacts(artifacts []string) ([]BuilderImagePair, error) {
	var pairs []BuilderImagePair
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
			pairs = append(pairs, BuilderImagePair{
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
			pair := BuilderImagePair{Builder: parsed.Payload, ImageName: a.Image}
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
			pair := BuilderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		case buildpacks.Name:
			parsed := struct {
				Payload buildpacks.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			pair := BuilderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		default:
			return nil, fmt.Errorf("unknown builder type in CLI artifacts: %q", a.Name)
		}
	}
	return pairs, nil
}

// For each image parsed from all k8s manifests, prompt the user for the builder that builds the referenced image
func ResolveBuilderImages(builderConfigs []InitBuilder, images []string, force bool) ([]BuilderImagePair, error) {
	// If nothing to choose, don't bother prompting
	if len(images) == 0 || len(builderConfigs) == 0 {
		return []BuilderImagePair{}, nil
	}

	// if we only have 1 image and 1 build config, don't bother prompting
	if len(images) == 1 && len(builderConfigs) == 1 {
		return []BuilderImagePair{{
			Builder:   builderConfigs[0],
			ImageName: images[0],
		}}, nil
	}

	if force {
		return nil, errors.New("unable to automatically resolve builder/image pairs; run `skaffold init` without `--force` to manually resolve ambiguities")
	}

	return resolveBuilderImagesInteractively(builderConfigs, images)
}

func resolveBuilderImagesInteractively(builderConfigs []InitBuilder, images []string) ([]BuilderImagePair, error) {
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
	pairs := []BuilderImagePair{}
	for {
		if len(images) == 0 {
			break
		}

		image := images[0]
		choice, err := prompt.BuildConfigFunc(image, append(choices, NoBuilder))
		if err != nil {
			return nil, err
		}

		if choice != NoBuilder {
			pairs = append(pairs, BuilderImagePair{Builder: choiceMap[choice], ImageName: image})
			choices = util.RemoveFromSlice(choices, choice)
		}
		images = util.RemoveFromSlice(images, image)
	}
	if len(choices) > 0 {
		logrus.Warnf("unused builder configs found in repository: %v", choices)
	}
	return pairs, nil
}

func StripTags(taggedImages []string) []string {
	// Remove tags from image names
	var images []string
	for _, image := range taggedImages {
		parsed, err := docker.ParseReference(image)
		if err != nil {
			// It's possible that it's a templatized name that can't be parsed as is.
			warnings.Printf("Couldn't parse image [%s]: %s", image, err.Error())
			continue
		}
		if parsed.Digest != "" {
			warnings.Printf("Ignoring image referenced by digest: [%s]", image)
			continue
		}

		images = append(images, parsed.BaseName)
	}
	return images
}
