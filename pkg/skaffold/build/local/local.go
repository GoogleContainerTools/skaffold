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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
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
	digest, err := b.runBuildForArtifact(ctx, out, artifact)
	if err != nil {
		return "", errors.Wrap(err, "build artifact")
	}

	if b.alreadyTagged == nil {
		b.alreadyTagged = make(map[string]string)
	}
	if tag, present := b.alreadyTagged[digest]; present {
		return tag, nil
	}

	tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, tag.Options{
		ImageName: artifact.ImageName,
		Digest:    digest,
	})
	if err != nil {
		return "", errors.Wrap(err, "generating tag")
	}

	if err := b.retagAndPush(ctx, out, digest, tag, artifact); err != nil {
		return "", errors.Wrap(err, "tagging")
	}

	b.alreadyTagged[digest] = tag

	return tag, nil
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact) (string, error) {
	switch {
	case artifact.DockerArtifact != nil:
		return b.buildDocker(ctx, out, artifact.Workspace, artifact.DockerArtifact)

	case artifact.BazelArtifact != nil:
		return b.buildBazel(ctx, out, artifact.Workspace, artifact)

	case artifact.JibMavenArtifact != nil:
		return b.buildJibMaven(ctx, out, artifact.Workspace, artifact)

	case artifact.JibGradleArtifact != nil:
		return b.buildJibGradle(ctx, out, artifact.Workspace, artifact)

	default:
		return "", fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}

func (b *Builder) retagAndPush(ctx context.Context, out io.Writer, digest string, newTag string, artifact *latest.Artifact) error {
	if b.pushImages && (artifact.JibMavenArtifact != nil || artifact.JibGradleArtifact != nil || artifact.BazelArtifact != nil) {
		// when pushing images, jib/bazel build them directly to the registry. all we need to do here is add a tag to the remote.

		// NOTE: the digest returned by the builders when in push mode is the digest of the remote image that was built to the registry.
		// when adding the tag to the remote, we need to specify the registry it was built to so go-containerregistry knows
		// where to look when grabbing the remote image reference.
		if err := docker.AddTag(fmt.Sprintf("%s@%s", artifact.ImageName, digest), newTag); err != nil {
			return errors.Wrap(err, "tagging image")
		}
		return nil
	}

	if err := b.localDocker.Tag(ctx, digest, newTag); err != nil {
		return err
	}

	if b.pushImages {
		if _, err := b.localDocker.Push(ctx, out, newTag); err != nil {
			return errors.Wrap(err, "pushing")
		}
	}

	return nil
}
