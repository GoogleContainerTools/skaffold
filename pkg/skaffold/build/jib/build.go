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

package jib

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

// Build builds an artifact with Jib.
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latestV2.Artifact, tag string, platforms platform.Matcher) (string, error) {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "jib",
		"Context":     instrumentation.PII(artifact.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	t, err := DeterminePluginType(ctx, artifact.Workspace, artifact.JibArtifact)
	if err != nil {
		return "", err
	}

	switch t {
	case JibMaven:
		if b.pushImages {
			return b.buildJibMavenToRegistry(ctx, out, artifact.Workspace, artifact.JibArtifact, artifact.Dependencies, tag, platforms)
		}
		return b.buildJibMavenToDocker(ctx, out, artifact.Workspace, artifact.JibArtifact, artifact.Dependencies, tag, platforms)

	case JibGradle:
		if b.pushImages {
			return b.buildJibGradleToRegistry(ctx, out, artifact.Workspace, artifact.JibArtifact, artifact.Dependencies, tag, platforms)
		}
		return b.buildJibGradleToDocker(ctx, out, artifact.Workspace, artifact.JibArtifact, artifact.Dependencies, tag, platforms)

	default:
		return "", unknownPluginType(artifact.Workspace)
	}
}

func (b *Builder) SupportedPlatforms() platform.Matcher { return platform.All }
