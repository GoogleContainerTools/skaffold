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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/bazel"
	buildahbuilder "github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildah"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/custom"
	dockerbuilder "github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/podman"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Builder uses the host docker daemon to build and tag the image.
type Builder struct {
	local latestV1.LocalBuild

	cfg                docker.Config
	localDocker        docker.LocalDaemon
	buildahClient      *podman.Buildah
	localCluster       bool
	pushImages         bool
	tryImportMissing   bool
	prune              bool
	pruneChildren      bool
	skipTests          bool
	mode               config.RunMode
	kubeContext        string
	builtImages        []string
	insecureRegistries map[string]bool
	muted              build.Muted
	localPruner        *pruner
	imageHandler       ImageHandler
	artifactStore      build.ArtifactStore
	sourceDependencies graph.SourceDependenciesCache
}

type Config interface {
	docker.Config

	GlobalConfig() string
	GetKubeContext() string
	GetCluster() config.Cluster
	SkipTests() bool
	Mode() config.RunMode
	NoPruneChildren() bool
	Muted() config.Muted
	PushImages() config.BoolOrUndefined
}

type BuilderContext interface {
	Config
	ArtifactStore() build.ArtifactStore
	SourceDependenciesResolver() graph.SourceDependenciesCache
}

// ImageHandler handles the real operations for the image
// currently implemented with docker and buildah
type ImageHandler interface {
	ListImages(ctx context.Context, name string) ([]imageSummary, error)
	// prune removes images from the handler by their ids and returns all pruned images
	Prune(ctx context.Context, ids []string, pruneChildren bool) ([]string, error)
	DiskUsage(ctx context.Context) (uint64, error)
	GetImageID(ctx context.Context, tag string) (string, error)
	TagWithImageID(ctx context.Context, tag string, imageID string) (string, error)
}

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(ctx context.Context, bCtx BuilderContext, buildCfg *latestV1.LocalBuild) (*Builder, error) {
	builder := &Builder{
		local:              *buildCfg,
		cfg:                bCtx,
		kubeContext:        bCtx.GetKubeContext(),
		skipTests:          bCtx.SkipTests(),
		mode:               bCtx.Mode(),
		prune:              bCtx.Prune(),
		pruneChildren:      !bCtx.NoPruneChildren(),
		insecureRegistries: bCtx.GetInsecureRegistries(),
		muted:              bCtx.Muted(),
		artifactStore:      bCtx.ArtifactStore(),
		sourceDependencies: bCtx.SourceDependenciesResolver(),
	}

	if buildCfg.Podman != nil {
		buildahClient, err := podman.NewBuildah()
		if err != nil {
			return nil, err
		}
		localBuildah := NewLocalBuildah(buildahClient)
		builder.localPruner = newPruner(localBuildah, !bCtx.NoPruneChildren())
		builder.buildahClient = buildahClient
		builder.imageHandler = localBuildah
	} else {
		localDocker, err := docker.NewAPIClient(ctx, bCtx)
		if err != nil {
			return nil, fmt.Errorf("getting docker client: %w", err)
		}
		d := NewLocalDocker(localDocker)
		builder.localPruner = newPruner(d, !bCtx.NoPruneChildren())
		builder.localDocker = localDocker
		builder.imageHandler = d
	}

	cluster := bCtx.GetCluster()
	pushFlag := bCtx.PushImages()

	var pushImages bool
	switch {
	case pushFlag.Value() != nil:
		pushImages = *pushFlag.Value()
		log.Entry(context.TODO()).Debugf("push value set via skaffold build --push flag, --push=%t", *pushFlag.Value())
	case buildCfg.Push == nil:
		pushImages = cluster.PushImages
		log.Entry(context.TODO()).Debugf("push value not present in NewBuilder, defaulting to %t because cluster.PushImages is %t", pushImages, cluster.PushImages)
	default:
		pushImages = *buildCfg.Push
	}

	builder.tryImportMissing = buildCfg.TryImportMissing
	builder.localCluster = cluster.Local
	builder.pushImages = pushImages
	return builder, nil
}

// Prune removes all images built with Skaffold
func (b *Builder) Prune(ctx context.Context, _ io.Writer) error {
	var toPrune []string
	seen := make(map[string]bool)

	for _, img := range b.builtImages {
		if !seen[img] && !b.localPruner.isPruned(img) {
			toPrune = append(toPrune, img)
			seen[img] = true
		}
	}

	b.localPruner.cleanup(ctx, true /*sync*/, toPrune)
	return nil
}

// artifactBuilder represents a per artifact builder interface
type artifactBuilder interface {
	Build(ctx context.Context, out io.Writer, a *latestV1.Artifact, tag string) (string, error)
}

// newPerArtifactBuilder returns an instance of `artifactBuilder`
func newPerArtifactBuilder(b *Builder, a *latestV1.Artifact) (artifactBuilder, error) {
	switch {
	case a.DockerArtifact != nil:
		return dockerbuilder.NewArtifactBuilder(b.localDocker, b.cfg, b.local.UseDockerCLI, b.local.UseBuildkit, b.pushImages, b.artifactStore, b.sourceDependencies), nil

	case a.BazelArtifact != nil:
		return bazel.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages), nil

	case a.JibArtifact != nil:
		return jib.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages, b.skipTests, b.artifactStore), nil

	case a.CustomArtifact != nil:
		// required artifacts as environment variables
		dependencies := util.EnvPtrMapToSlice(docker.ResolveDependencyImages(a.Dependencies, b.artifactStore, true), "=")
		return custom.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages, b.skipTests, append(b.retrieveExtraEnv(), dependencies...)), nil

	case a.BuildpackArtifact != nil:
		return buildpacks.NewArtifactBuilder(b.localDocker, b.pushImages, b.mode, b.artifactStore), nil

	case a.KoArtifact != nil:
		return ko.NewArtifactBuilder(b.localDocker, b.pushImages, b.mode, b.insecureRegistries), nil

	case a.BuildahArtifact != nil:
		return buildahbuilder.NewBuilder(b.buildahClient, b.pushImages), nil
	default:
		return nil, fmt.Errorf("unexpected type %q for local artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}
