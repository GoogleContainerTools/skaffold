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

	"golang.org/x/sync/errgroup"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type ArtifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)

type scheduler struct {
	artifacts       []*latest.Artifact
	nodes           []node // size len(artifacts)
	artifactBuilder ArtifactBuilder
	logger          logAggregator
	results         ArtifactStore
	concurrencySem  countingSemaphore
}

func newScheduler(artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int, out io.Writer, store ArtifactStore) *scheduler {
	s := scheduler{
		artifacts:       artifacts,
		nodes:           createNodes(artifacts),
		artifactBuilder: artifactBuilder,
		logger:          newLogAggregator(out, len(artifacts), concurrency),
		results:         store,
		concurrencySem:  newCountingSemaphore(concurrency),
	}
	return &s
}

func (s *scheduler) run(ctx context.Context, tags tag.ImageTags) ([]Artifact, error) {
	g, gCtx := errgroup.WithContext(ctx)

	for i := range s.artifacts {
		i := i

		// Create a goroutine for each element in dag. Each goroutine waits on its dependencies to finish building.
		// Because our artifacts form a DAG, at least one of the goroutines should be able to start building.
		// Wrap in an error group so that all other builds are cancelled as soon as any one fails.
		g.Go(func() error {
			return s.build(gCtx, tags, i)
		})
	}
	// print output for all artifact builds in order
	s.logger.PrintInOrder(gCtx)
	if err := g.Wait(); err != nil {
		event.BuildSequenceFailed(err)
		return nil, err
	}
	return s.results.GetArtifacts(s.artifacts)
}

func (s *scheduler) build(ctx context.Context, tags tag.ImageTags, i int) error {
	n := s.nodes[i]
	a := s.artifacts[i]
	err := n.waitForDependencies(ctx)
	if err != nil {
		// `waitForDependencies` only returns `context.Canceled` error
		event.BuildCanceled(a.ImageName)
		return err
	}
	release := s.concurrencySem.acquire()
	defer release()

	event.BuildInProgress(a.ImageName)

	w, closeFn, err := s.logger.GetWriter()
	if err != nil {
		event.BuildFailed(a.ImageName, err)
		return err
	}
	defer closeFn()

	finalTag, err := performBuild(ctx, w, tags, a, s.artifactBuilder)
	if err != nil {
		event.BuildFailed(a.ImageName, err)
		return err
	}

	s.results.Record(a, finalTag)
	n.markComplete()
	event.BuildComplete(a.ImageName)
	return nil
}

// InOrder builds a list of artifacts in dependency order.
func InOrder(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int, store ArtifactStore) ([]Artifact, error) {
	// `concurrency` specifies the max number of builds that can run at any one time. If concurrency is 0, then all builds can run in parallel.
	if concurrency == 0 {
		concurrency = len(artifacts)
	}
	if concurrency > 1 {
		color.Default.Fprintf(out, "Building %d artifacts in parallel\n", concurrency)
	}
	s := newScheduler(artifacts, artifactBuilder, concurrency, out, store)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return s.run(ctx, tags)
}

func performBuild(ctx context.Context, cw io.Writer, tags tag.ImageTags, artifact *latest.Artifact, build ArtifactBuilder) (string, error) {
	color.Default.Fprintf(cw, "Building [%s]...\n", artifact.ImageName)
	tag, present := tags[artifact.ImageName]
	if !present {
		return "", fmt.Errorf("unable to find tag for image %s", artifact.ImageName)
	}
	return build(ctx, cw, artifact, tag)
}
