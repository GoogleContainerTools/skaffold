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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/custom"
	dockerbuilder "github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
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
}

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(cfg Config, buildCfg *latest.LocalBuild) (*Builder, error) {
	localDocker, err := docker.NewAPIClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("getting docker client: %w", err)
	}

	cluster := cfg.GetCluster()

	var pushImages bool
	if buildCfg.Push == nil {
		pushImages = cluster.PushImages
		logrus.Debugf("push value not present, defaulting to %t because cluster.PushImages is %t", pushImages, cluster.PushImages)
	} else {
		pushImages = *buildCfg.Push
	}

	tryImportMissing := buildCfg.TryImportMissing

	return &Builder{
		local:              *buildCfg,
		cfg:                cfg,
		kubeContext:        cfg.GetKubeContext(),
		localDocker:        localDocker,
		localCluster:       cluster.Local,
		pushImages:         pushImages,
		tryImportMissing:   tryImportMissing,
		skipTests:          cfg.SkipTests(),
		mode:               cfg.Mode(),
		prune:              cfg.Prune(),
		pruneChildren:      !cfg.NoPruneChildren(),
		localPruner:        newPruner(localDocker, !cfg.NoPruneChildren()),
		insecureRegistries: cfg.GetInsecureRegistries(),
		muted:              cfg.Muted(),
	}, nil
}

func (b *Builder) ArtifactStore(store build.ArtifactStore) {
	b.artifactStore = store
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
	_, err := b.localDocker.Prune(ctx, toPrune, b.pruneChildren)
	return err
}

// artifactBuilder represents a per artifact builder interface
type artifactBuilder interface {
	Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error)
}

// newPerArtifactBuilder returns an instance of `artifactBuilder`
func newPerArtifactBuilder(b *Builder, a *latest.Artifact) (artifactBuilder, error) {
	switch {
	case a.DockerArtifact != nil:
		return dockerbuilder.NewArtifactBuilder(b.localDocker, b.local.UseDockerCLI, b.local.UseBuildkit, b.pushImages, b.prune, b.cfg.Mode(), b.cfg.GetInsecureRegistries(), b.artifactStore), nil

	case a.BazelArtifact != nil:
		return bazel.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages), nil

	case a.JibArtifact != nil:
		return jib.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages, b.skipTests, b.artifactStore), nil

	case a.CustomArtifact != nil:
		// required artifacts as environment variables
		dependencies := util.EnvPtrMapToSlice(docker.ResolveDependencyImages(a.Dependencies, b.artifactStore, true), "=")
		return custom.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages, append(b.retrieveExtraEnv(), dependencies...)), nil

	case a.BuildpackArtifact != nil:
		return buildpacks.NewArtifactBuilder(b.localDocker, b.pushImages, b.mode, b.artifactStore), nil

	default:
		return nil, fmt.Errorf("unexpected type %q for local artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}
