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

package cmd

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	tag "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag/util"
)

// DefaultRepoFn takes an image tag and returns either a new tag with the default repo prefixed, or the original tag if
// no default repo is specified.
type DefaultRepoFn func(string) (string, error)

func getBuildArtifactsAndSetTags(artifacts []*latest.Artifact, defaulterFn DefaultRepoFn) ([]graph.Artifact, error) {
	buildArtifacts, err := mergeBuildArtifacts(fromBuildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts(), artifacts)
	if err != nil {
		return nil, err
	}

	return applyDefaultRepoToArtifacts(buildArtifacts, defaulterFn)
}

func applyDefaultRepoToArtifacts(artifacts []graph.Artifact, defaulterFn DefaultRepoFn) ([]graph.Artifact, error) {
	for i := range artifacts {
		updatedTag, err := defaulterFn(artifacts[i].Tag)
		if err != nil {
			return nil, err
		}
		artifacts[i].Tag = updatedTag
	}

	return artifacts, nil
}

func mergeBuildArtifacts(fromFile, fromCLI []graph.Artifact, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	var buildArtifacts []graph.Artifact
	for _, artifact := range artifacts {
		buildArtifacts = append(buildArtifacts, graph.Artifact{
			ImageName:   artifact.ImageName,
			RuntimeType: artifact.RuntimeType,
		})
	}

	// Tags provided by file take precedence over those provided on the command line
	buildArtifacts = build.MergeWithPreviousBuilds(fromCLI, buildArtifacts)
	buildArtifacts = build.MergeWithPreviousBuilds(fromFile, buildArtifacts)

	buildArtifacts, err := applyCustomTag(buildArtifacts)
	if err != nil {
		return nil, err
	}

	// Check that every image has a non empty tag
	if err := validateArtifactTags(buildArtifacts); err != nil {
		return nil, err
	}

	return buildArtifacts, nil
}

func applyCustomTag(artifacts []graph.Artifact) ([]graph.Artifact, error) {
	if opts.CustomTag != "" {
		var result []graph.Artifact
		for _, artifact := range artifacts {
			if artifact.Tag == "" {
				artifact.Tag = artifact.ImageName + ":" + opts.CustomTag
			} else {
				newTag, err := tag.SetImageTag(artifact.Tag, opts.CustomTag)
				if err != nil {
					return nil, err
				}
				artifact.Tag = newTag
			}
			result = append(result, artifact)
		}
		return result, nil
	}
	return artifacts, nil
}

func validateArtifactTags(artifacts []graph.Artifact) error {
	for _, artifact := range artifacts {
		if artifact.Tag == "" {
			return fmt.Errorf("no tag provided for image [%s]", artifact.ImageName)
		}
	}
	return nil
}
