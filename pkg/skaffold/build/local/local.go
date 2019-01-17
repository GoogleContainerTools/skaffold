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

package local

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if b.localCluster {
		color.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.kubeContext)
	}
	defer b.localDocker.Close()

	// TODO(dgageot): parallel builds
	return build.InSequence(ctx, out, tagger, artifacts, b.buildArtifact)
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
	tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, tag.Options{
		ImageName: artifact.ImageName,
	})
	if err != nil {
		return "", errors.Wrap(err, "generating tag")
	}

	digestOrImageID, err := b.runBuildForArtifact(ctx, out, artifact, tag)
	if err != nil {
		return "", errors.Wrap(err, "build artifact")
	}

	if !b.pushImages {
		imageID := digestOrImageID
		return strings.TrimPrefix(imageID, "sha256:"), nil
	}

	// Jib pushes images directly
	if artifact.JibMavenArtifact != nil || artifact.JibGradleArtifact != nil {
		digest := digestOrImageID
		return tag + "@" + digest, nil
	}

	digest, err := b.localDocker.Push(ctx, out, tag)
	if err != nil {
		return "", errors.Wrap(err, "pushing")
	}
	return tag + "@" + digest, nil
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	switch {
	case artifact.DockerArtifact != nil:
		return b.buildDocker(ctx, out, artifact.Workspace, artifact.DockerArtifact, tag)

	case artifact.BazelArtifact != nil:
		return b.buildBazel(ctx, out, artifact.Workspace, artifact.BazelArtifact, tag)

	case artifact.JibMavenArtifact != nil:
		return b.buildJibMaven(ctx, out, artifact.Workspace, artifact.JibMavenArtifact, tag)

	case artifact.JibGradleArtifact != nil:
		return b.buildJibGradle(ctx, out, artifact.Workspace, artifact.JibGradleArtifact, tag)

	default:
		return "", fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}
