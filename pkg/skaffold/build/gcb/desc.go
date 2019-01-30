/*
Copyright 2018 The Skaffold Authors

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
	"errors"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func (b *Builder) buildDescription(artifact *latest.Artifact, bucket, object string) (*cloudbuild.Build, error) {
	steps, err := b.buildSteps(artifact)
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
		Images: []string{artifact.ImageName},
		Options: &cloudbuild.BuildOptions{
			DiskSizeGb:  b.DiskSizeGb,
			MachineType: b.MachineType,
		},
		Timeout: b.Timeout,
	}, nil
}

func (b *Builder) buildSteps(artifact *latest.Artifact) ([]*cloudbuild.BuildStep, error) {
	switch {
	case artifact.DockerArtifact != nil:
		return b.dockerBuildSteps(artifact.ImageName, artifact.DockerArtifact), nil

	case artifact.BazelArtifact != nil:
		return nil, errors.New("skaffold can't build a bazel artifact with Google Cloud Build")

	case artifact.JibMavenArtifact != nil:
		return b.jibMavenBuildSteps(artifact.ImageName, artifact.JibMavenArtifact), nil

	case artifact.JibGradleArtifact != nil:
		return b.jibGradleBuildSteps(artifact.ImageName, artifact.JibGradleArtifact), nil

	default:
		return nil, fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}
