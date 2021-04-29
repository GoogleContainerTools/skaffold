/*
Copyright 2021 The Skaffold Authors

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

package graph

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/custom"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// for testing
var getDependenciesFunc = sourceDependenciesForArtifact

// SourceDependenciesCache provides an interface to evaluate and cache the source dependencies for artifacts.
// This additionally includes the source dependencies from all other artifacts that are in the transitive closure of its artifact dependencies.
type SourceDependenciesCache interface {
	// TransitiveArtifactDependencies returns the source dependencies for the target artifact, including the source dependencies from all other artifacts that are in the transitive closure of its artifact dependencies.
	// The result (even if an error) is cached so that the function is evaluated only once for every artifact. The cache is reset before the start of the next devloop.
	TransitiveArtifactDependencies(ctx context.Context, a *latest_v1.Artifact) ([]string, error)
	// SingleArtifactDependencies returns the source dependencies for only the target artifact.
	// The result (even if an error) is cached so that the function is evaluated only once for every artifact. The cache is reset before the start of the next devloop.
	SingleArtifactDependencies(ctx context.Context, a *latest_v1.Artifact) ([]string, error)
	// Reset removes the cached source dependencies for all artifacts
	Reset()
}

func NewSourceDependenciesCache(cfg docker.Config, r docker.ArtifactResolver, g ArtifactGraph) SourceDependenciesCache {
	return &dependencyResolverImpl{cfg: cfg, artifactResolver: r, artifactGraph: g, cache: util.NewSyncStore()}
}

type dependencyResolverImpl struct {
	cfg              docker.Config
	artifactResolver docker.ArtifactResolver
	artifactGraph    ArtifactGraph
	cache            *util.SyncStore
}

func (r *dependencyResolverImpl) TransitiveArtifactDependencies(ctx context.Context, a *latest_v1.Artifact) ([]string, error) {
	deps, err := r.SingleArtifactDependencies(ctx, a)
	if err != nil {
		return nil, err
	}
	for _, ad := range a.Dependencies {
		d, err := r.TransitiveArtifactDependencies(ctx, r.artifactGraph[ad.ImageName])
		if err != nil {
			return nil, err
		}
		deps = append(deps, d...)
	}
	return deps, nil
}

func (r *dependencyResolverImpl) SingleArtifactDependencies(ctx context.Context, a *latest_v1.Artifact) ([]string, error) {
	res := r.cache.Exec(a.ImageName, func() interface{} {
		d, e := getDependenciesFunc(ctx, a, r.cfg, r.artifactResolver)
		if e != nil {
			return e
		}
		return d
	})
	if err, ok := res.(error); ok {
		return nil, err
	}
	return res.([]string), nil
}

func (r *dependencyResolverImpl) Reset() {
	r.cache = util.NewSyncStore()
}

// sourceDependenciesForArtifact returns the build dependencies for the current artifact.
func sourceDependenciesForArtifact(ctx context.Context, a *latest_v1.Artifact, cfg docker.Config, r docker.ArtifactResolver) ([]string, error) {
	var (
		paths []string
		err   error
	)

	switch {
	case a.DockerArtifact != nil:
		// Required artifacts cannot be resolved when `ResolveDependencyImages` runs prior to a completed build sequence (like `skaffold build` or the first iteration of `skaffold dev`).
		// However it only affects the behavior for Dockerfiles with ONBUILD instructions, and there's no functional change even for those scenarios.
		// For single build scenarios like `build` and `run`, it is called for the cache hash calculations which are already handled in `artifactHasher`.
		// For `dev` it will succeed on the first dev loop and list any additional dependencies found from the base artifact's ONBUILD instructions as a file added instead of modified (see `filemon.Events`)
		deps := docker.ResolveDependencyImages(a.Dependencies, r, false)
		args, evalErr := docker.EvalBuildArgs(cfg.Mode(), a.Workspace, a.DockerArtifact.DockerfilePath, a.DockerArtifact.BuildArgs, deps)
		if evalErr != nil {
			return nil, fmt.Errorf("unable to evaluate build args: %w", evalErr)
		}
		paths, err = docker.GetDependencies(ctx, docker.NewBuildConfig(a.Workspace, a.ImageName, a.DockerArtifact.DockerfilePath, args), cfg)

	case a.KanikoArtifact != nil:
		deps := docker.ResolveDependencyImages(a.Dependencies, r, false)
		args, evalErr := docker.EvalBuildArgs(cfg.Mode(), a.Workspace, a.KanikoArtifact.DockerfilePath, a.KanikoArtifact.BuildArgs, deps)
		if evalErr != nil {
			return nil, fmt.Errorf("unable to evaluate build args: %w", evalErr)
		}
		paths, err = docker.GetDependencies(ctx, docker.NewBuildConfig(a.Workspace, a.ImageName, a.KanikoArtifact.DockerfilePath, args), cfg)

	case a.BazelArtifact != nil:
		paths, err = bazel.GetDependencies(ctx, a.Workspace, a.BazelArtifact)

	case a.JibArtifact != nil:
		paths, err = jib.GetDependencies(ctx, a.Workspace, a.JibArtifact)

	case a.CustomArtifact != nil:
		paths, err = custom.GetDependencies(ctx, a.Workspace, a.ImageName, a.CustomArtifact, cfg)

	case a.BuildpackArtifact != nil:
		paths, err = buildpacks.GetDependencies(ctx, a.Workspace, a.BuildpackArtifact)

	default:
		return nil, fmt.Errorf("unexpected artifact type %q:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}

	if err != nil {
		return nil, err
	}

	return util.AbsolutePaths(a.Workspace, paths), nil
}
