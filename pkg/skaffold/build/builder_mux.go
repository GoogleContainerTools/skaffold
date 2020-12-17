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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type BuilderMux struct {
	builders    []PipelineBuilder
	byImageName map[string]PipelineBuilder
	store       ArtifactStore
	concurrency int
}

func NewBuilderMux(runCtx *runcontext.RunContext, store ArtifactStore, builder func(p latest.Pipeline) (PipelineBuilder, error)) (Builder, error) {
	pipelines := runCtx.GetPipelines()
	m := make(map[string]PipelineBuilder)
	var sl []PipelineBuilder
	minConcurrency := -1
	for _, p := range pipelines {
		b, err := builder(p)
		if err != nil {
			return nil, fmt.Errorf("creating builder: %w", err)
		}
		sl = append(sl, b)
		for _, a := range p.Build.Artifacts {
			m[a.ImageName] = b
		}
		concurrency := b.Concurrency()
		if minConcurrency < 0 {
			minConcurrency = concurrency
		} else if concurrency > 0 && concurrency < minConcurrency {
			// set mux concurrency to be the minimum of all builders' concurrency. (concurrency = 0 means unlimited)
			minConcurrency = concurrency
		}
	}
	if minConcurrency > len(m) {
		minConcurrency = 0 // if specified concurrency is greater than maximum number of parallel jobs, then just set it to unlimited
	}

	return &BuilderMux{builders: sl, byImageName: m, store: store, concurrency: minConcurrency}, nil
}

func (b *BuilderMux) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error) {
	m := make(map[PipelineBuilder]bool)
	for _, a := range artifacts {
		m[b.byImageName[a.ImageName]] = true
	}

	for builder := range m {
		if err := builder.PreBuild(ctx, out); err != nil {
			return nil, err
		}
	}

	builder := func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
		p := b.byImageName[artifact.ImageName]
		artifactBuilder := p.Build(ctx, out, artifact)
		return artifactBuilder(ctx, out, artifact, tag)
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

func (b *BuilderMux) Prune(ctx context.Context, writer io.Writer) error {
	for _, builder := range b.builders {
		if err := builder.Prune(ctx, writer); err != nil {
			return err
		}
	}
	return nil
}
