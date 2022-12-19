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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
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
			d.generatedArtifactInfos = append(d.generatedArtifactInfos, getGeneratedArtifactInfo(d.builders[0]))
			return nil
		}
		// we already have the image, just use it and return
		if len(d.unresolvedImages) == 1 {
			d.artifactInfos = append(d.artifactInfos, ArtifactInfo{
				Builder:   d.builders[0],
				ImageName: d.unresolvedImages[0],
			})
			return nil
		}
	}

	if d.force {
		return d.resolveBuilderImagesForcefully()
	}

	return d.resolveBuilderImagesInteractively()
}

func (d *defaultBuildInitializer) resolveBuilderImagesForcefully() error {
	// In the case of 1 image and multiple builders, respects the ordering Docker > Jib > Bazel > Buildpacks
	if len(d.unresolvedImages) == 1 {
		image := d.unresolvedImages[0]
		choice := d.builders[0]
		for _, builder := range d.builders {
			if builderRank(builder) < builderRank(choice) {
				choice = builder
			}
		}

		d.artifactInfos = append(d.artifactInfos, ArtifactInfo{Builder: choice, ImageName: image})
		d.unresolvedImages = []string{}
		return nil
	}

	return errors.BuilderImageAmbiguitiesErr{}
}

func builderRank(builder InitBuilder) int {
	a := builder.ArtifactType("")
	switch {
	case a.DockerArtifact != nil:
		return 1
	case a.JibArtifact != nil:
		return 2
	case a.KoArtifact != nil:
		return 3
	case a.BazelArtifact != nil:
		return 4
	case a.BuildpackArtifact != nil:
		return 5
	}

	return 6
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
			d.artifactInfos = append(d.artifactInfos, ArtifactInfo{Builder: choiceMap[choice], ImageName: image})
			choices = stringslice.Remove(choices, choice)
		}
		d.unresolvedImages = stringslice.Remove(d.unresolvedImages, image)
	}
	if len(choices) > 0 {
		chosen, err := prompt.ChooseBuildersFunc(choices)
		if err != nil {
			return err
		}

		for _, choice := range chosen {
			d.generatedArtifactInfos = append(d.generatedArtifactInfos, getGeneratedArtifactInfo(choiceMap[choice]))
		}
	}
	return nil
}

func getGeneratedArtifactInfo(b InitBuilder) GeneratedArtifactInfo {
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
	return GeneratedArtifactInfo{
		ArtifactInfo: ArtifactInfo{
			Builder:   b,
			ImageName: sanitizeImageName(imageName),
		},
		ManifestPath: filepath.Join(path, "deployment.yaml"),
	}
}

func sanitizeImageName(imageName string) string {
	// Replace unsupported characters with `_`
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9-_]`).ReplaceAllString(imageName, `-`)

	// Truncate to 128 characters
	if len(sanitized) > 128 {
		return sanitized[0:128]
	}

	return sanitized
}
