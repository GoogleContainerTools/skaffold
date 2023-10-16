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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/custom"
	dockerbuilder "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/ko"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// Builder uses the host docker daemon to build and tag the image.
type Builder struct {
	local latest.LocalBuild

	cfg                docker.Config
	localDocker        docker.LocalDaemon
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
func NewBuilder(ctx context.Context, bCtx BuilderContext, buildCfg *latest.LocalBuild) (*Builder, error) {
	localDocker, err := docker.NewAPIClient(ctx, bCtx)
	if err != nil {
		return nil, fmt.Errorf("getting docker client: %w", err)
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
		localCluster:       cluster.Local,
		pushImages:         pushImages,
		tryImportMissing:   tryImportMissing,
		skipTests:          bCtx.SkipTests(),
		mode:               bCtx.Mode(),
		prune:              bCtx.Prune(),
		pruneChildren:      !bCtx.NoPruneChildren(),
		localPruner:        newPruner(localDocker, !bCtx.NoPruneChildren()),
		insecureRegistries: bCtx.GetInsecureRegistries(),
		muted:              bCtx.Muted(),
		artifactStore:      bCtx.ArtifactStore(),
		sourceDependencies: bCtx.SourceDependenciesResolver(),
	}, nil
}

// artifactBuilder represents a per artifact builder interface
type artifactBuilder interface {
	Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, platforms platform.Matcher) (string, error)
	SupportedPlatforms() platform.Matcher
}

// newPerArtifactBuilder returns an instance of `artifactBuilder`
func newPerArtifactBuilder(b *Builder, a *latest.Artifact) (artifactBuilder, error) {
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

	default:
		return nil, fmt.Errorf("unexpected type %q for local artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}
