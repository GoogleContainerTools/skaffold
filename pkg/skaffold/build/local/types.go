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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildah"
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
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/containers/common/libimage"
)

// Builder uses the host docker daemon to build and tag the image.
type Builder struct {
	local latestV1.LocalBuild

	cfg                docker.Config
	localDocker        docker.LocalDaemon
	libImageRuntime    *libimage.Runtime
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

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(ctx context.Context, bCtx BuilderContext, buildCfg *latestV1.LocalBuild) (*Builder, error) {
	var localDocker docker.LocalDaemon
	var err error
	var libimageRuntime *libimage.Runtime
	if buildCfg.UseBuildah {
		libimageRuntime, err = buildah.NewLibImageRuntime()
	} else {
		localDocker, err = docker.NewAPIClient(ctx, bCtx)
		if err != nil {
			return nil, fmt.Errorf("getting docker client: %w", err)
		}
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

	tryImportMissing := buildCfg.TryImportMissing

	return &Builder{
		local:              *buildCfg,
		cfg:                bCtx,
		kubeContext:        bCtx.GetKubeContext(),
		localDocker:        localDocker,
		libImageRuntime:    libimageRuntime,
		localCluster:       cluster.Local,
		pushImages:         pushImages,
		tryImportMissing:   tryImportMissing,
		skipTests:          bCtx.SkipTests(),
		mode:               bCtx.Mode(),
		prune:              bCtx.Prune(),
		pruneChildren:      !bCtx.NoPruneChildren(),
		localPruner:        newPruner(buildCfg.UseBuildah, libimageRuntime, localDocker, !bCtx.NoPruneChildren()),
		insecureRegistries: bCtx.GetInsecureRegistries(),
		muted:              bCtx.Muted(),
		artifactStore:      bCtx.ArtifactStore(),
		sourceDependencies: bCtx.SourceDependenciesResolver(),
	}, nil
}

// Prune uses the docker API client to remove all images built with Skaffold
func (b *Builder) Prune(ctx context.Context, _ io.Writer) error {
	var toPrune []string
	seen := make(map[string]bool)

	for _, img := range b.builtImages {
		if !seen[img] && !b.localPruner.isPruned(img) {
			toPrune = append(toPrune, img)
			seen[img] = true
		}
	}
	if b.local.UseBuildah {
		_, buildErrs := b.libImageRuntime.RemoveImages(ctx, toPrune, &libimage.RemoveImagesOptions{})
		if len(buildErrs) > 0 {
			var errors []string
			for _, buildErr := range buildErrs {
				errors = append(errors, buildErr.Error())
			}
			return fmt.Errorf("buildah pruning images: %v", strings.Join(errors, ";"))
		}
		return nil
	}
	_, err := b.localDocker.Prune(ctx, toPrune, b.pruneChildren)
	return err
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
		return buildah.NewBuilder(b.pushImages), nil
	default:
		return nil, fmt.Errorf("unexpected type %q for local artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}
