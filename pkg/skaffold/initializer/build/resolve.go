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
	"sort"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// For each image parsed from all k8s manifests, prompt the user for the builder that builds the referenced image
func (d *defaultBuildInitializer) resolveBuilderImages() error {
	// If nothing to choose, don't bother prompting
	if len(d.unresolvedImages) == 0 || len(d.builders) == 0 {
		return nil
	}

	// if we only have 1 image and 1 build config, don't bother prompting
	if len(d.unresolvedImages) == 1 && len(d.builders) == 1 {
		d.builderImagePairs = append(d.builderImagePairs, BuilderImagePair{
			Builder:   d.builders[0],
			ImageName: d.unresolvedImages[0],
		})
		return nil
	}

	if d.force {
		return errors.New("unable to automatically resolve builder/image pairs; run `skaffold init` without `--force` to manually resolve ambiguities")
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
	pairs := []BuilderImagePair{}
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
			pairs = append(pairs, BuilderImagePair{Builder: choiceMap[choice], ImageName: image})
			choices = util.RemoveFromSlice(choices, choice)
		}
		d.unresolvedImages = util.RemoveFromSlice(d.unresolvedImages, image)
	}
	if len(choices) > 0 {
		logrus.Warnf("unused builder configs found in repository: %v", choices)
	}
	d.builderImagePairs = append(d.builderImagePairs, pairs...)
	return nil
}
