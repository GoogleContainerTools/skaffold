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
	"path/filepath"
	"sort"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	tag "github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
)

func matchBuildersToImages(builders []InitBuilder, images []string) ([]ArtifactInfo, []InitBuilder, []string) {
	images = tag.StripTags(images, true)

	var artifactInfos []ArtifactInfo
	var unresolvedImages = make(sortedSet)
	for _, image := range images {
		builderIdx := findExactlyOneMatchingBuilder(builders, image)

		// exactly one builder found for the image
		if builderIdx != -1 {
			// save the pair
			artifactInfos = append(artifactInfos, ArtifactInfo{ImageName: image, Builder: builders[builderIdx]})
			// remove matched builder from builderConfigs
			builders = append(builders[:builderIdx], builders[builderIdx+1:]...)
		} else {
			// No definite pair found, add to images list
			unresolvedImages.add(image)
		}
	}
	return artifactInfos, builders, unresolvedImages.values()
}

func findExactlyOneMatchingBuilder(builderConfigs []InitBuilder, image string) int {
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

// ResolveBuilderInteractively resolve builder for an image and returns a map of the image to the chosen builder
// It also returns builders that were not chosen in this process.
func ResolveBuilderInteractively(builders []InitBuilder, unresolvedImages []string) (map[string]InitBuilder, []InitBuilder, error) {
	chosen := map[string]InitBuilder{}
	choices, choiceMap := buildChoiceMap(builders)

	// For each choice, use prompt string to pair builder config with k8s image
	for {
		if len(unresolvedImages) == 0 {
			break
		}

		image := unresolvedImages[0]
		choice, err := prompt.BuildConfigFunc(image, append(choices, NoBuilder))
		if err != nil {
			return nil, nil, err
		}

		if choice != NoBuilder {
			chosen[image] = choiceMap[choice]
			choices = stringslice.Remove(choices, choice)
		}
		unresolvedImages = stringslice.Remove(unresolvedImages, image)
	}
	unusedBuilder := []InitBuilder{}
	for _, k := range choices {
		unusedBuilder = append(unusedBuilder, choiceMap[k])
	}
	return chosen, unusedBuilder, nil
}

func buildChoiceMap(builders []InitBuilder) ([]string, map[string]InitBuilder) {
	choices := make([]string, len(builders))

	choiceMap := make(map[string]InitBuilder, len(builders))
	for i, buildConfig := range builders {
		choice := buildConfig.Describe()
		choices[i] = choice
		choiceMap[choice] = buildConfig
	}
	sort.Strings(choices)
	return choices, choiceMap
}

// Artifacts takes builder image pairs and workspaces and creates a list of latest.Artifacts from the data.
func Artifacts(artifactInfos []ArtifactInfo) []*latest.Artifact {
	var artifacts []*latest.Artifact

	for _, info := range artifactInfos {
		// Don't create artifact build config for "None" builder
		if info.Builder.Name() == NoneBuilderName {
			continue
		}
		workspace := info.Workspace
		if workspace == "" {
			workspace = filepath.Dir(info.Builder.Path())
		}
		artifact := &latest.Artifact{
			ImageName:    info.ImageName,
			ArtifactType: info.Builder.ArtifactType(workspace),
		}

		if workspace != "." {
			// to make skaffold.yaml more portable across OS-es we should always generate /-delimited filepaths
			artifact.Workspace = filepath.ToSlash(workspace)
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts
}
