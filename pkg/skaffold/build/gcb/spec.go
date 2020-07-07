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

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
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
	buildSpec.Options.WorkerPool = b.WorkerPool
	buildSpec.Options.Logging = b.Logging
	buildSpec.Options.LogStreamingOption = b.LogStreamingOption
	buildSpec.Timeout = b.Timeout

	return buildSpec, nil
}

func (b *Builder) buildSpecForArtifact(a *latest.Artifact, tag string) (cloudbuild.Build, error) {
	switch {
	case a.KanikoArtifact != nil:
		return b.kanikoBuildSpec(a.KanikoArtifact, tag)

	case a.DockerArtifact != nil:
		return b.dockerBuildSpec(a.DockerArtifact, tag)

	case a.JibArtifact != nil:
		return b.jibBuildSpec(a, tag)

	case a.BuildpackArtifact != nil:
		return b.buildpackBuildSpec(a.BuildpackArtifact, tag)

	default:
		return cloudbuild.Build{}, fmt.Errorf("unexpected type %q for gcb artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}
