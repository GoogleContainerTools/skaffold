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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Build builds a list of artifacts with Kaniko.
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latest.Artifact) build.ArtifactBuilder {
	builder := build.WithLogFile(b.buildArtifact, b.cfg.Muted())
	return builder
}

func (b *Builder) PreBuild(ctx context.Context, out io.Writer) error {
	teardownPullSecret, err := b.setupPullSecret(ctx, out)
	if err != nil {
		return fmt.Errorf("setting up pull secret: %w", err)
	}
	b.teardownFunc = append(b.teardownFunc, teardownPullSecret)

	if b.DockerConfig != nil {
		teardownDockerConfigSecret, err := b.setupDockerConfigSecret(ctx, out)
		if err != nil {
			return fmt.Errorf("setting up docker config secret: %w", err)
		}
		b.teardownFunc = append(b.teardownFunc, teardownDockerConfigSecret)
	}
	return nil
}

func (b *Builder) PostBuild(_ context.Context, _ io.Writer) error {
	for _, f := range b.teardownFunc {
		f()
	}
	return nil
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	// TODO: [#4922] Implement required artifact resolution from the `artifactStore`
	digest, err := b.runBuildForArtifact(ctx, out, artifact, tag)
	if err != nil {
		return "", err
	}

	return build.TagWithDigest(tag, digest), nil
}

func (b *Builder) Concurrency() int {
	return b.ClusterDetails.Concurrency
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	// required artifacts as build-args
	requiredImages := docker.ResolveDependencyImages(a.Dependencies, b.artifactStore, true)
	switch {
	case a.KanikoArtifact != nil:
		return b.buildWithKaniko(ctx, out, a.Workspace, a.ImageName, a.KanikoArtifact, tag, requiredImages)

	case a.CustomArtifact != nil:
		return custom.NewArtifactBuilder(nil, b.cfg, true, append(b.retrieveExtraEnv(), util.EnvPtrMapToSlice(requiredImages, "=")...)).Build(ctx, out, a, tag)

	default:
		return "", fmt.Errorf("unexpected type %q for in-cluster artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}

func (b *Builder) retrieveExtraEnv() []string {
	env := []string{
		fmt.Sprintf("%s=%s", constants.KubeContext, b.cfg.GetKubeContext()),
		fmt.Sprintf("%s=%s", constants.Namespace, b.ClusterDetails.Namespace),
		fmt.Sprintf("%s=%s", constants.PullSecretName, b.ClusterDetails.PullSecretName),
		fmt.Sprintf("%s=%s", constants.Timeout, b.ClusterDetails.Timeout),
	}
	if b.ClusterDetails.DockerConfig != nil {
		env = append(env, fmt.Sprintf("%s=%s", constants.DockerConfigSecretName, b.ClusterDetails.DockerConfig.SecretName))
	}
	return env
}
