/*
Copyright 2020 The Skaffold Authors

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

package build

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
)

// BuilderMux encapsulates multiple build configs.
type BuilderMux struct {
	builders    []PipelineBuilder
	byImageName map[string]PipelineBuilder
	store       ArtifactStore
	concurrency int
	buildx      bool
	cache       Cache
}

type Cache interface {
	AddArtifact(ctx context.Context, a graph.Artifact) error
}

// Config represents an interface for getting all config pipelines.
type Config interface {
	GetPipelines() []latest.Pipeline
	DefaultRepo() *string
	Mode() config.RunMode
	MultiLevelRepo() *bool
	GlobalConfig() string
	BuildConcurrency() int
}

// NewBuilderMux returns an implementation of `build.BuilderMux`.
func NewBuilderMux(cfg Config, store ArtifactStore, cache Cache, builder func(p latest.Pipeline) (PipelineBuilder, error)) (*BuilderMux, error) {
	pipelines := cfg.GetPipelines()
	m := make(map[string]PipelineBuilder)
	var pbs []PipelineBuilder
	for _, p := range pipelines {
		b, err := builder(p)
		if err != nil {
			return nil, fmt.Errorf("creating builder: %w", err)
		}
		pbs = append(pbs, b)
		for _, a := range p.Build.Artifacts {
			m[a.ImageName] = b
		}
	}
	concurrency := getConcurrency(pbs, cfg.BuildConcurrency())
	buildx := config.GetDetectBuildX(cfg.GlobalConfig())
	return &BuilderMux{builders: pbs, byImageName: m, store: store, concurrency: concurrency, cache: cache, buildx: buildx}, nil
}

// Build executes the specific image builder for each artifact in the given artifact slice.
func (b *BuilderMux) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, resolver platform.Resolver, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	m := make(map[PipelineBuilder]bool)
	for _, a := range artifacts {
		m[b.byImageName[a.ImageName]] = true
	}

	for builder := range m {
		if err := builder.PreBuild(ctx, out); err != nil {
			return nil, err
		}
	}

	builderF := func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string, platforms platform.Matcher) (string, error) {
		p := b.byImageName[artifact.ImageName]
		pl, err := filterBuildEnvSupportedPlatforms(p.SupportedPlatforms(), platforms)
		if err != nil {
			return "", err
		}
		platforms = pl

		artifactBuilder := p.Build(ctx, out, artifact)
		hooksOpts, err := hooks.NewBuildEnvOpts(artifact, tag, p.PushImages())
		if err != nil {
			return "", err
		}
		r := hooks.BuildRunner(artifact.LifecycleHooks, hooksOpts)
		if err = r.RunPreHooks(ctx, out); err != nil {
			return "", err
		}
		var built string

		// buildx creates multiplatform images via buildkit directly
		if platforms.IsMultiPlatform() && !SupportsMultiPlatformBuild(*artifact) && !b.buildx {
			built, err = CreateMultiPlatformImage(ctx, out, artifact, tag, platforms, artifactBuilder)
		} else {
			built, err = artifactBuilder(ctx, out, artifact, tag, platforms)
		}

		if err != nil {
			return "", err
		}

		if err := b.cache.AddArtifact(ctx, graph.Artifact{
			ImageName:   artifact.ImageName,
			Tag:         built,
			RuntimeType: artifact.RuntimeType,
		}); err != nil {
			log.Entry(ctx).Warnf("error adding artifact to cache; caching may not work as expected: %v", err)
		}

		if err = r.RunPostHooks(ctx, out); err != nil {
			return "", err
		}

		return built, nil
	}

	err := checkMultiplatformHaveRegistry(b, artifacts, resolver)
	if err != nil {
		return nil, fmt.Errorf("%w", err) // TODO: remove error wrapping after fixing #7790
	}

	ar, err := InOrder(ctx, out, tags, resolver, artifacts, builderF, b.concurrency, b.store)
	if err != nil {
		return nil, err
	}

	for builder := range m {
		if errB := builder.PostBuild(ctx, out); errB != nil {
			return nil, errB
		}
	}

	return ar, nil
}

// Prune removes built images.
func (b *BuilderMux) Prune(ctx context.Context, writer io.Writer) error {
	for _, builder := range b.builders {
		if err := builder.Prune(ctx, writer); err != nil {
			return err
		}
	}
	return nil
}

// filterBuildEnvSupportedPlatforms filters the target platforms to those supported by the selected build environment (local/googleCloudBuild/cluster).
func filterBuildEnvSupportedPlatforms(supported platform.Matcher, target platform.Matcher) (platform.Matcher, error) {
	if target.IsEmpty() {
		return target, nil
	}
	pl := target.Intersect(supported)
	if pl.IsEmpty() {
		return platform.Matcher{}, fmt.Errorf("target build platforms %q not supported by current build environment. Supported platforms: %q", target, supported)
	}
	return pl, nil
}

func getConcurrency(pbs []PipelineBuilder, cliConcurrency int) int {
	ctx := context.TODO()
	if cliConcurrency >= 0 {
		log.Entry(ctx).Infof("build concurrency set to cli concurrency %d", cliConcurrency)
		return cliConcurrency
	}
	minConcurrency := -1
	for i, b := range pbs {
		if b.Concurrency() == nil {
			continue
		}
		concurrency := *b.Concurrency()

		// set mux concurrency to be the minimum of all builders' concurrency. (concurrency = 0 means unlimited)
		switch {
		case minConcurrency < 0:
			minConcurrency = concurrency
			log.Entry(ctx).Infof("build concurrency first set to %d parsed from %s[%d]", minConcurrency, reflect.TypeOf(b).String(), i)
		case concurrency > 0 && (minConcurrency == 0 || concurrency < minConcurrency):
			minConcurrency = concurrency
			log.Entry(ctx).Infof("build concurrency updated to %d parsed from %s[%d]", minConcurrency, reflect.TypeOf(b).String(), i)
		default:
			log.Entry(ctx).Infof("build concurrency value %d parsed from %s[%d] is ignored since it's not less than previously set value %d", concurrency, reflect.TypeOf(b).String(), i, minConcurrency)
		}
	}
	if minConcurrency < 0 {
		log.Entry(ctx).Infof("build concurrency set to default value of %d", minConcurrency)
		// set default concurrency to 1 for local builder. For GCB and Cluster build the default value is 0
		return constants.DefaultLocalConcurrency
	}
	log.Entry(ctx).Infof("final build concurrency value is %d", minConcurrency)
	return minConcurrency
}

func checkMultiplatformHaveRegistry(b *BuilderMux, artifacts []*latest.Artifact, platforms platform.Resolver) error {
	for _, artifact := range artifacts {
		pb := b.byImageName[artifact.ImageName]
		hasExternalRegistry := pb.PushImages()
		pl, err := filterBuildEnvSupportedPlatforms(pb.SupportedPlatforms(), platforms.GetPlatforms(artifact.ImageName))
		if err != nil {
			return err
		}

		if pl.IsMultiPlatform() && !hasExternalRegistry {
			return noRegistryForMultiplatformBuildErr(fmt.Errorf("multi-platform build requires pushing images to a valid container registry"))
		}
	}

	return nil
}
