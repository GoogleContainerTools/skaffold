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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func matchBuildersToImages(builders []InitBuilder, images []string) ([]ArtifactInfo, []InitBuilder, []string) {
	images = tag.StripTags(images)

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

// Artifacts takes builder image pairs and workspaces and creates a list of latest.Artifacts from the data.
func Artifacts(artifactInfos []ArtifactInfo) []*latest.Artifact {
	var artifacts []*latest.Artifact

	for _, info := range artifactInfos {
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
