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

package local

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (b *Builder) Build(ctx context.Context, out io.Writer, a *latestV2.Artifact) build.ArtifactBuilder {
	if b.prune {
		b.localPruner.asynchronousCleanupOldImages(ctx, []string{a.ImageName})
	}
	builder := build.WithLogFile(b.buildArtifact, b.muted)
	return builder
}

func (b *Builder) PreBuild(_ context.Context, out io.Writer) error {
	if b.localCluster {
		output.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.kubeContext)
	}
	return nil
}

func (b *Builder) PostBuild(ctx context.Context, _ io.Writer) error {
	defer b.localDocker.Close()
	if b.prune {
		if b.mode == config.RunModes.Build {
			b.localPruner.synchronousCleanupOldImages(ctx, b.builtImages)
		} else {
			b.localPruner.asynchronousCleanupOldImages(ctx, b.builtImages)
		}
	}
	return nil
}

func (b *Builder) Concurrency() *int { return b.local.Concurrency }

func (b *Builder) PushImages() bool {
	return b.pushImages
}

func (b *Builder) SupportedPlatforms() platform.Matcher { return platform.All }

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, a *latestV2.Artifact, tag string, platforms platform.Matcher) (string, error) {
	digestOrImageID, err := b.runBuildForArtifact(ctx, out, a, tag, platforms)
	if err != nil {
		return "", err
	}

	if b.pushImages {
		// only track images for pruning when building with docker
		// if we're pushing a bazel image, it was built directly to the registry
		if a.DockerArtifact != nil {
			imageID, err := b.getImageIDForTag(ctx, tag)
			if err != nil {
				log.Entry(ctx).Warn("unable to inspect image: built images may not be cleaned up correctly by skaffold")
			}
			if imageID != "" {
				b.builtImages = append(b.builtImages, imageID)
			}
		}

		digest := digestOrImageID
		return build.TagWithDigest(tag, digest), nil
	}

	imageID := digestOrImageID
	b.builtImages = append(b.builtImages, imageID)
	return build.TagWithImageID(ctx, tag, imageID, b.localDocker)
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, a *latestV2.Artifact, tag string, platforms platform.Matcher) (string, error) {
	if !b.pushImages {
		// All of the builders will rely on a local Docker:
		// + Either to build the image,
		// + Or to docker load it.
		// Let's fail fast if Docker is not available
		if _, err := b.localDocker.ServerVersion(ctx); err != nil {
			return "", err
		}
	}

	builder, err := newPerArtifactBuilder(b, a)
	if err != nil {
		return "", err
	}
	if platforms.IsNotEmpty() {
		supported := builder.SupportedPlatforms()
		if p := platforms.Intersect(supported); p.IsNotEmpty() {
			platforms = p
		} else {
			return "", fmt.Errorf("builder for artifact %q doesn't support building for target platforms: %q. Supported platforms are %q", a.ImageName, platforms, supported)
		}
	}
	return builder.Build(ctx, out, a, tag, platforms)
}

func (b *Builder) getImageIDForTag(ctx context.Context, tag string) (string, error) {
	insp, _, err := b.localDocker.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		return "", err
	}
	return insp.ID, nil
}

func (b *Builder) retrieveExtraEnv() []string {
	return b.localDocker.ExtraEnv()
}
