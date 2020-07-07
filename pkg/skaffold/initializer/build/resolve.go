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
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// For each image parsed from all k8s manifests, prompt the user for the builder that builds the referenced image
func (d *defaultBuildInitializer) resolveBuilderImages() error {
	// If nothing to choose, don't bother prompting
	if len(d.builders) == 0 {
		return nil
	}

	// if there's only one builder config, no need to prompt
	if len(d.builders) == 1 {
		if len(d.unresolvedImages) == 0 {
			// no image was parsed from k8s manifests, so we create an image name
			d.generatedBuilderImagePairs = append(d.generatedBuilderImagePairs, getGeneratedBuilderPair(d.builders[0]))
			return nil
		}
		// we already have the image, just use it and return
		if len(d.unresolvedImages) == 1 {
			d.builderImagePairs = append(d.builderImagePairs, BuilderImagePair{
				Builder:   d.builders[0],
				ImageName: d.unresolvedImages[0],
			})
			return nil
		}
	}

	if d.force {
		return errors.BuilderImageAmbiguitiesErr{}
	}

	return d.resolveBuilderImagesInteractively()
}

func (d *defaultBuildInitializer) resolveBuilderImagesInteractively() error {
	// Build map from choice string to builder config struct
	choices := make([]string, len(d.builders))
	choiceMap := make(map[string]InitBuilder, len(d.builders))
	for i, buildConfig := range d.builders {
		choice := buildConfig.Describe()
		choices[i] = choice
		choiceMap[choice] = buildConfig
	}
	sort.Strings(choices)

	// For each choice, use prompt string to pair builder config with k8s image
	for {
		if len(d.unresolvedImages) == 0 {
			break
		}

		image := d.unresolvedImages[0]
		choice, err := prompt.BuildConfigFunc(image, append(choices, NoBuilder))
		if err != nil {
			return err
		}

		if choice != NoBuilder {
			d.builderImagePairs = append(d.builderImagePairs, BuilderImagePair{Builder: choiceMap[choice], ImageName: image})
			choices = util.RemoveFromSlice(choices, choice)
		}
		d.unresolvedImages = util.RemoveFromSlice(d.unresolvedImages, image)
	}
	if len(choices) > 0 {
		// TODO(nkubala): should we ask user if they want to generate here?
		for _, choice := range choices {
			d.generatedBuilderImagePairs = append(d.generatedBuilderImagePairs, getGeneratedBuilderPair(choiceMap[choice]))
		}
	}
	return nil
}

func getGeneratedBuilderPair(b InitBuilder) GeneratedBuilderImagePair {
	path := b.Path()
	var imageName string
	// if the builder is in a nested directory, use that as the image name AND the path to write the manifest
	// otherwise, use the builder as the image name itself, and the current directory to write the manifest
	if filepath.Dir(path) != "." {
		imageName = strings.ToLower(filepath.Dir(path))
		path = imageName
	} else {
		imageName = fmt.Sprintf("%s-image", strings.ToLower(path))
		path = "."
	}
	return GeneratedBuilderImagePair{
		BuilderImagePair: BuilderImagePair{
			Builder:   b,
			ImageName: sanitizeImageName(imageName),
		},
		ManifestPath: filepath.Join(path, "deployment.yaml"),
	}
}

func sanitizeImageName(imageName string) string {
	// Replace unsupported characters with `_`
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9-._]`).ReplaceAllString(imageName, `-`)

	// Truncate to 128 characters
	if len(sanitized) > 128 {
		return sanitized[0:128]
	}

	return sanitized
}
