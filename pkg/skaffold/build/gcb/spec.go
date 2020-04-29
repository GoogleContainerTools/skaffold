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
	"errors"
	"fmt"

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
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
	if buildSpec.Options == nil {
		buildSpec.Options = &cloudbuild.BuildOptions{}
	}
	buildSpec.Options.DiskSizeGb = b.DiskSizeGb
	buildSpec.Options.MachineType = b.MachineType
	buildSpec.Options.Logging = b.Logging
	buildSpec.Options.LogStreamingOption = b.LogStreamingOption
	buildSpec.Timeout = b.Timeout

	return buildSpec, nil
}

func (b *Builder) buildSpecForArtifact(artifact *latest.Artifact, tag string) (cloudbuild.Build, error) {
	switch {
	case artifact.KanikoArtifact != nil:
		return b.kanikoBuildSpec(artifact.KanikoArtifact, tag)

	case artifact.DockerArtifact != nil:
		return b.dockerBuildSpec(artifact.DockerArtifact, tag)

	case artifact.JibArtifact != nil:
		return b.jibBuildSpec(artifact, tag)

	case artifact.BazelArtifact != nil:
		return cloudbuild.Build{}, errors.New("skaffold can't build a bazel artifact with Google Cloud Build")

	case artifact.CustomArtifact != nil:
		return cloudbuild.Build{}, errors.New("skaffold can't build a custom artifact with Google Cloud Build")

	case artifact.BuildpackArtifact != nil:
		return b.buildpackBuildSpec(artifact.BuildpackArtifact, tag)

	default:
		return cloudbuild.Build{}, fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}
