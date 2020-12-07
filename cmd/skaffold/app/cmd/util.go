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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func getBuildArtifactsAndSetTags(out io.Writer, r runner.Runner, config *latest.SkaffoldConfig) ([]build.Artifact, error) {
	buildArtifacts, err := getArtifacts(out, fromBuildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts(), config.Build.Artifacts)
	if err != nil {
		return nil, err
	}

	for i := range buildArtifacts {
		tag, err := r.ApplyDefaultRepo(buildArtifacts[i].Tag)
		if err != nil {
			return nil, err
		}
		buildArtifacts[i].Tag = tag
	}

	return buildArtifacts, nil
}

func getArtifacts(out io.Writer, fromFile, fromCLI []build.Artifact, artifacts []*latest.Artifact) ([]build.Artifact, error) {
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
