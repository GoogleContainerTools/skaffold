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
	"context"
	"fmt"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

func (b *Builder) buildSpec(ctx context.Context, artifact *latestV2.Artifact, tag string, platforms platform.Matcher, bucket, object string) (cloudbuild.Build, error) {
	// Artifact specific build spec
	buildSpec, err := b.buildSpecForArtifact(ctx, artifact, tag, platforms)
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
	if b.WorkerPool != "" {
		buildSpec.Options.Pool = &cloudbuild.PoolOption{Name: b.WorkerPool}
	}
	buildSpec.Options.Logging = b.Logging
	buildSpec.Options.LogStreamingOption = b.LogStreamingOption
	buildSpec.Timeout = b.Timeout

	return buildSpec, nil
}

func (b *Builder) buildSpecForArtifact(ctx context.Context, a *latestV2.Artifact, tag string, platforms platform.Matcher) (cloudbuild.Build, error) {
	switch {
	case a.KanikoArtifact != nil:
		return b.kanikoBuildSpec(a, tag)

	case a.DockerArtifact != nil:
		return b.dockerBuildSpec(a, tag, platforms)

	case a.JibArtifact != nil:
		return b.jibBuildSpec(ctx, a, tag, platforms)

	case a.BuildpackArtifact != nil:
		// TODO: Buildpacks only supports building for platform linux/amd64. See https://github.com/GoogleCloudPlatform/buildpacks/issues/112
		if platforms.IsNotEmpty() && platforms.Intersect(platform.Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}}).IsEmpty() {
			return cloudbuild.Build{}, fmt.Errorf("buildpacks builder doesn't support building for platforms %s. Cannot build gcb artifact:\n%s", platforms.String(), misc.FormatArtifact(a))
		}
		return b.buildpackBuildSpec(a.BuildpackArtifact, tag, a.Dependencies)

	default:
		return cloudbuild.Build{}, fmt.Errorf("unexpected type %q for gcb artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}
