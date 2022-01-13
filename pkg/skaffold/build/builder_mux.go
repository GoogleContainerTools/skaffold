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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
)

// BuilderMux encapsulates multiple build configs.
type BuilderMux struct {
	builders    []PipelineBuilder
	byImageName map[string]PipelineBuilder
	store       ArtifactStore
	concurrency int
}

// Config represents an interface for getting all config pipelines.
type Config interface {
	GetPipelines() []latestV1.Pipeline
	DefaultRepo() *string
	MultiLevelRepo() *bool
	GlobalConfig() string
	BuildConcurrency() int
}

// NewBuilderMux returns an implementation of `build.BuilderMux`.
func NewBuilderMux(cfg Config, store ArtifactStore, builder func(p latestV1.Pipeline) (PipelineBuilder, error)) (*BuilderMux, error) {
	pipelines := cfg.GetPipelines()
	m := make(map[string]PipelineBuilder)
	var pb []PipelineBuilder
	minConcurrency := -1
	for i, p := range pipelines {
		b, err := builder(p)
		if err != nil {
			return nil, fmt.Errorf("creating builder: %w", err)
		}
		pb = append(pb, b)
		for _, a := range p.Build.Artifacts {
			m[a.ImageName] = b
		}

		if cfg.BuildConcurrency() >= 0 {
			minConcurrency = cfg.BuildConcurrency()
		} else {
			concurrency := b.Concurrency()
			// set mux concurrency to be the minimum of all builders' concurrency. (concurrency = 0 means unlimited)
			switch {
			case minConcurrency < 0:
				minConcurrency = concurrency
				log.Entry(context.TODO()).Infof("build concurrency first set to %d parsed from %s[%d]", minConcurrency, reflect.TypeOf(b).String(), i)
			case concurrency > 0 && (minConcurrency == 0 || concurrency < minConcurrency):
				minConcurrency = concurrency
				log.Entry(context.TODO()).Infof("build concurrency updated to %d parsed from %s[%d]", minConcurrency, reflect.TypeOf(b).String(), i)
			default:
				log.Entry(context.TODO()).Infof("build concurrency value %d parsed from %s[%d] is ignored since it's not less than previously set value %d", concurrency, reflect.TypeOf(b).String(), i, minConcurrency)
			}
		}
	}
	log.Entry(context.TODO()).Infof("final build concurrency value is %d", minConcurrency)

	return &BuilderMux{builders: pb, byImageName: m, store: store, concurrency: minConcurrency}, nil
}

// Build executes the specific image builder for each artifact in the given artifact slice.
func (b *BuilderMux) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latestV1.Artifact) ([]graph.Artifact, error) {
	m := make(map[PipelineBuilder]bool)
	for _, a := range artifacts {
		m[b.byImageName[a.ImageName]] = true
	}

	for builder := range m {
		if err := builder.PreBuild(ctx, out); err != nil {
			return nil, err
		}
	}

	builder := func(ctx context.Context, out io.Writer, artifact *latestV1.Artifact, tag string) (string, error) {
		p := b.byImageName[artifact.ImageName]
		artifactBuilder := p.Build(ctx, out, artifact)
		hooksOpts, err := hooks.NewBuildEnvOpts(artifact, tag, p.PushImages())
		if err != nil {
			return "", err
		}
		r := hooks.BuildRunner(artifact.LifecycleHooks, hooksOpts)
		var built string
		if err = r.RunPreHooks(ctx, out); err != nil {
			return "", err
		}
		if built, err = artifactBuilder(ctx, out, artifact, tag); err != nil {
			return "", err
		}
		if err = r.RunPostHooks(ctx, out); err != nil {
			return "", err
		}
		return built, nil
	}
	ar, err := InOrder(ctx, out, tags, artifacts, builder, b.concurrency, b.store)
	if err != nil {
		return nil, err
	}

	for builder := range m {
		if err := builder.PostBuild(ctx, out); err != nil {
			return nil, err
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
