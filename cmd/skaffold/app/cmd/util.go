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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func getBuildArtifactsAndSetTags(r runner.Runner, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	buildArtifacts, err := mergeBuildArtifacts(fromBuildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts(), artifacts)
	if err != nil {
		return nil, err
	}

	for _, artifact := range buildArtifacts {
		tag, err := r.ApplyDefaultRepo(artifact.Tag)
		if err != nil {
			return nil, err
		}
		artifact.Tag = tag
	}

	return buildArtifacts, nil
}

func mergeBuildArtifacts(fromFile, fromCLI []build.Artifact, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var buildArtifacts []build.Artifact
	for _, artifact := range artifacts {
		buildArtifacts = append(buildArtifacts, build.Artifact{
			ImageName: artifact.ImageName,
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

func applyCustomTag(artifacts []build.Artifact) ([]build.Artifact, error) {
	if opts.CustomTag != "" {
		var result []build.Artifact
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

func validateArtifactTags(artifacts []build.Artifact) error {
	for _, artifact := range artifacts {
		if artifact.Tag == "" {
			return fmt.Errorf("no tag provided for image [%s]", artifact.ImageName)
		}
	}
	return nil
}
