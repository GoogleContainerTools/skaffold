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

package kaniko

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// Build builds a list of artifacts with Kaniko.
func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	teardownPullSecret, err := b.setupPullSecret(out)
	if err != nil {
		return nil, errors.Wrap(err, "setting up pull secret")
	}
	defer teardownPullSecret()

	if b.DockerConfig != nil {
		teardownDockerConfigSecret, err := b.setupDockerConfigSecret(out)
		if err != nil {
			return nil, errors.Wrap(err, "setting up docker config secret")
		}
		defer teardownDockerConfigSecret()
	}

	return build.InParallel(ctx, out, tagger, artifacts, b.buildArtifactWithKaniko)
}

func (b *Builder) buildArtifactWithKaniko(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
	initialTag, err := b.run(ctx, out, artifact)
	if err != nil {
		return "", errors.Wrapf(err, "kaniko build for [%s]", artifact.ImageName)
	}

	digest, err := docker.RemoteDigest(initialTag)
	if err != nil {
		return "", errors.Wrap(err, "getting digest")
	}

	tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, tag.Options{
		ImageName: artifact.ImageName,
		Digest:    digest,
	})
	if err != nil {
		return "", errors.Wrap(err, "generating tag")
	}

	if err := docker.AddTag(initialTag, tag); err != nil {
		return "", errors.Wrap(err, "tagging image")
	}

	return tag, nil
}
