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

package cluster

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/custom"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// Build builds a list of artifacts with Kaniko.
func (b *Builder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
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

	var kanikoArtifacts []*latest.Artifact
	var otherArtifacts []*latest.Artifact

	for _, a := range artifacts {
		if a.ArtifactType.KanikoArtifact != nil {
			kanikoArtifacts = append(kanikoArtifacts, a)
			continue
		}
		otherArtifacts = append(otherArtifacts, a)
	}

	pArts, err := build.InParallel(ctx, out, tags, kanikoArtifacts, b.buildArtifactWithKaniko)
	if err != nil {
		return nil, errors.Wrap(err, "building artifacts in parallel with kaniko")
	}

	sArts, err := build.InSequence(ctx, out, tags, otherArtifacts, b.runBuildForArtifact)
	return append(pArts, sArts...), err
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	switch {
	case artifact.KanikoArtifact != nil:
		return b.buildArtifactWithKaniko(ctx, out, artifact, tag)

	case artifact.CustomArtifact != nil:
		return b.buildArtifactWithCustomBuilder(ctx, out, artifact, tag)

	default:
		return "", fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
	}
}

func (b *Builder) buildArtifactWithCustomBuilder(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	extraEnv := []string{fmt.Sprintf("%s=%s", constants.Kubecontext, b.kubeContext)}
	customArtifactBuilder := custom.NewArtifactBuilder(true, extraEnv)
	if err := customArtifactBuilder.Build(ctx, out, artifact, tag); err != nil {
		return "", errors.Wrapf(err, "building custom artifact %s", artifact.ImageName)
	}
	return docker.RemoteDigest(tag, b.insecureRegistries)
}

func (b *Builder) buildArtifactWithKaniko(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	digest, err := b.runKanikoBuild(ctx, out, artifact, tag)
	if err != nil {
		return "", errors.Wrapf(err, "kaniko build for [%s]", artifact.ImageName)
	}

	return tag + "@" + digest, nil
}
