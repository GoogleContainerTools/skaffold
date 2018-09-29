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
		if _, err := color.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.kubeContext); err != nil {
			return nil, errors.Wrap(err, "writing status")
		}
	}
	defer b.api.Close()

	// TODO(dgageot): parallel builds
	return build.InSequence(ctx, out, tagger, artifacts, b.buildArtifact)
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
	initialTag, err := b.runBuildForArtifact(ctx, out, artifact)
	if err != nil {
		return "", errors.Wrap(err, "build artifact")
	}

	digest, err := docker.Digest(ctx, b.api, initialTag)
	if err != nil {
		return "", errors.Wrapf(err, "getting digest: %s", initialTag)
	}
	if digest == "" {
		return "", fmt.Errorf("digest not found")
	}

	if b.alreadyTagged == nil {
		b.alreadyTagged = make(map[string]string)
	}
	if tag, present := b.alreadyTagged[digest]; present {
		return tag, nil
	}

	tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, &tag.Options{
		ImageName: artifact.ImageName,
		Digest:    digest,
	})
	if err != nil {
		return "", errors.Wrap(err, "generating tag")
	}

	if err := b.api.ImageTag(ctx, initialTag, tag); err != nil {
		return "", errors.Wrap(err, "tagging")
	}

	if b.pushImages {
		if err := docker.RunPush(ctx, b.api, tag, out); err != nil {
			return "", errors.Wrap(err, "pushing")
		}
	}

	b.alreadyTagged[digest] = tag

	return tag, nil
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact) (string, error) {
	switch {
	case artifact.DockerArtifact != nil:
		return b.buildDocker(ctx, out, artifact.Workspace, artifact.DockerArtifact)

	case artifact.BazelArtifact != nil:
		return b.buildBazel(ctx, out, artifact.Workspace, artifact.BazelArtifact)

	default:
		return "", fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}
