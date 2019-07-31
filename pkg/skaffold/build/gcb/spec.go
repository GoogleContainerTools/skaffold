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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func (b *Builder) buildSpec(artifact *latest.Artifact, tag, bucket, object string) (cloudbuild.Build, error) {
	// Artifact specific build spec
	buildSpec, err := b.buildSpecForArtifact(artifact, tag)
	if err != nil {
		return buildSpec, err
	}

	// Common build spec
	buildSpec.LogsBucket = bucket
	buildSpec.Source = &cloudbuild.Source{
		StorageSource: &cloudbuild.StorageSource{
			Bucket: bucket,
			Object: object,
		},
	}
	buildSpec.Options = &cloudbuild.BuildOptions{
		DiskSizeGb:  b.DiskSizeGb,
		MachineType: b.MachineType,
	}
	buildSpec.Timeout = b.Timeout

	return buildSpec, nil
}

func (b *Builder) buildSpecForArtifact(artifact *latest.Artifact, tag string) (cloudbuild.Build, error) {
	switch {
	case artifact.DockerArtifact != nil:
		return b.dockerBuildSpec(artifact.DockerArtifact, tag)

	case artifact.JibMavenArtifact != nil:
		return b.jibMavenBuildSpec(artifact.JibMavenArtifact, tag), nil

	case artifact.JibGradleArtifact != nil:
		return b.jibGradleBuildSpec(artifact.JibGradleArtifact, tag), nil

	case artifact.BazelArtifact != nil:
		return cloudbuild.Build{}, errors.New("skaffold can't build a bazel artifact with Google Cloud Build")

	default:
		return cloudbuild.Build{}, fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}
