/*
Copyright 2019 The Skaffold Authors

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

package gcb

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func (b *Builder) buildDescription(artifact *latest.Artifact, tag, bucket, object string) (*cloudbuild.Build, error) {
	tags := []string{tag}
	if artifact.WorkspaceHash != "" {
		tags = append(tags, cache.HashTag(artifact))
	}

	steps, err := b.buildSteps(artifact, tags)
	if err != nil {
		return nil, err
	}

	return &cloudbuild.Build{
		LogsBucket: bucket,
		Source: &cloudbuild.Source{
			StorageSource: &cloudbuild.StorageSource{
				Bucket: bucket,
				Object: object,
			},
		},
		Steps:  steps,
		Images: tags,
		Options: &cloudbuild.BuildOptions{
			DiskSizeGb:  b.DiskSizeGb,
			MachineType: b.MachineType,
		},
		Timeout: b.Timeout,
	}, nil
}

func (b *Builder) buildSteps(artifact *latest.Artifact, tags []string) ([]*cloudbuild.BuildStep, error) {
	switch {
	case artifact.DockerArtifact != nil:
		return b.dockerBuildSteps(artifact.DockerArtifact, tags)

	case artifact.BazelArtifact != nil:
		return nil, errors.New("skaffold can't build a bazel artifact with Google Cloud Build")

		// TODO: build multiple tagged images with jib in GCB (priyawadhwa@)
	case artifact.JibMavenArtifact != nil:
		return b.jibMavenBuildSteps(artifact.JibMavenArtifact, tags[0]), nil

	case artifact.JibGradleArtifact != nil:
		return b.jibGradleBuildSteps(artifact.JibGradleArtifact, tags[0]), nil

	default:
		return nil, fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}
