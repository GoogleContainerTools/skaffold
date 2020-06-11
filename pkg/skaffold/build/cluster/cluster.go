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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Build builds a list of artifacts with Kaniko.
func (b *Builder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	teardownPullSecret, err := b.setupPullSecret(out)
	if err != nil {
		return nil, fmt.Errorf("setting up pull secret: %w", err)
	}
	defer teardownPullSecret()

	if b.DockerConfig != nil {
		teardownDockerConfigSecret, err := b.setupDockerConfigSecret(out)
		if err != nil {
			return nil, fmt.Errorf("setting up docker config secret: %w", err)
		}
		defer teardownDockerConfigSecret()
	}

	return build.InParallel(ctx, out, tags, artifacts, b.buildArtifact, b.ClusterDetails.Concurrency)
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	digest, err := b.runBuildForArtifact(ctx, out, artifact, tag)
	if err != nil {
		return "", err
	}

	return build.TagWithDigest(tag, digest), nil
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	switch {
	case a.KanikoArtifact != nil:
		return b.buildWithKaniko(ctx, out, a.Workspace, a.KanikoArtifact, tag)

	case a.CustomArtifact != nil:
		return custom.NewArtifactBuilder(nil, b.insecureRegistries, true, b.retrieveExtraEnv()).Build(ctx, out, a, tag)

	default:
		return "", fmt.Errorf("unexpected type %q for in-cluster artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}

func (b *Builder) retrieveExtraEnv() []string {
	env := []string{
		fmt.Sprintf("%s=%s", constants.KubeContext, b.kubeContext),
		fmt.Sprintf("%s=%s", constants.Namespace, b.ClusterDetails.Namespace),
		fmt.Sprintf("%s=%s", constants.PullSecretName, b.ClusterDetails.PullSecretName),
		fmt.Sprintf("%s=%s", constants.Timeout, b.ClusterDetails.Timeout),
	}
	if b.ClusterDetails.DockerConfig != nil {
		env = append(env, fmt.Sprintf("%s=%s", constants.DockerConfigSecretName, b.ClusterDetails.DockerConfig.SecretName))
	}
	return env
}
