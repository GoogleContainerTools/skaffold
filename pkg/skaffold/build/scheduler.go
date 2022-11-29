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
	"errors"
	"fmt"
	"io"
	"strconv"

	"golang.org/x/sync/errgroup"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
)

type ArtifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string, platforms platform.Matcher) (string, error)

type scheduler struct {
	artifacts       []*latest.Artifact
	nodes           []node // size len(artifacts)
	artifactBuilder ArtifactBuilder
	logger          logAggregator
	results         ArtifactStore
	concurrencySem  countingSemaphore
	reportFailure   bool
}

func newScheduler(artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int, out io.Writer, store ArtifactStore) *scheduler {
	s := scheduler{
		artifacts:       artifacts,
		nodes:           createNodes(artifacts),
		artifactBuilder: artifactBuilder,
		logger:          newLogAggregator(out, len(artifacts), concurrency),
		results:         store,
		concurrencySem:  newCountingSemaphore(concurrency),

		// avoid visual stutters from reporting failures inline and Skaffold's final command output
		reportFailure: concurrency != 1 && len(artifacts) > 1,
	}
	return &s
}

func (s *scheduler) run(ctx context.Context, tags tag.ImageTags, platforms platform.Resolver) ([]graph.Artifact, error) {
	g, gCtx := errgroup.WithContext(ctx)

	for i := range s.artifacts {
		i := i

		// Create a goroutine for each element in dag. Each goroutine waits on its dependencies to finish building.
		// Because our artifacts form a DAG, at least one of the goroutines should be able to start building.
		// Wrap in an error group so that all other builds are cancelled as soon as any one fails.
		g.Go(func() error {
			return s.build(gCtx, tags, platforms, i)
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

func (s *scheduler) build(ctx context.Context, tags tag.ImageTags, platforms platform.Resolver, i int) error {
	n := s.nodes[i]
	a := s.artifacts[i]
	err := n.waitForDependencies(ctx)
	if err != nil {
		// `waitForDependencies` only returns `context.Canceled` error
		event.BuildCanceled(a.ImageName, platforms.GetPlatforms(a.ImageName).String())
		eventV2.BuildCanceled(a.ImageName, platforms.GetPlatforms(a.ImageName).String(), err)
		return err
	}
	release := s.concurrencySem.acquire()
	defer release()

	event.BuildInProgress(a.ImageName, platforms.GetPlatforms(a.ImageName).String())
	eventV2.BuildInProgress(a.ImageName, platforms.GetPlatforms(a.ImageName).String())
	ctx, endTrace := instrumentation.StartTrace(ctx, "build_BuildInProgress", map[string]string{
		"ArtifactNumber": strconv.Itoa(i),
		"ImageName":      instrumentation.PII(a.ImageName),
	})
	defer endTrace()

	w, closeFn, err := s.logger.GetWriter(ctx)
	if err != nil {
		event.BuildFailed(a.ImageName, platforms.GetPlatforms(a.ImageName).String(), err)
		eventV2.BuildFailed(a.ImageName, platforms.GetPlatforms(a.ImageName).String(), err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	defer closeFn()

	w, ctx = output.WithEventContext(ctx, w, constants.Build, a.ImageName)
	output.Default.Fprintf(w, "Building [%s]...\n", a.ImageName)
	finalTag, err := performBuild(ctx, w, tags, platforms, a, s.artifactBuilder)
	if err != nil {
		event.BuildFailed(a.ImageName, platforms.GetPlatforms(a.ImageName).String(), err)
		endTrace(instrumentation.TraceEndError(err))
		if errors.Is(ctx.Err(), context.Canceled) {
			output.Yellow.Fprintf(w, "Build [%s] was canceled\n", a.ImageName)
			eventV2.BuildCanceled(a.ImageName, platforms.GetPlatforms(a.ImageName).String(), err)
			return err
		}
		if s.reportFailure {
			output.Red.Fprintf(w, "Build [%s] failed: %v\n", a.ImageName, err)
		}
		eventV2.BuildFailed(a.ImageName, platforms.GetPlatforms(a.ImageName).String(), err)
		return fmt.Errorf("build [%s] failed: %w", a.ImageName, err)
	}

	output.Default.Fprintf(w, "Build [%s] succeeded\n", a.ImageName)
	s.results.Record(a, finalTag)
	n.markComplete()
	event.BuildComplete(a.ImageName, platforms.GetPlatforms(a.ImageName).String())
	eventV2.BuildSucceeded(a.ImageName, platforms.GetPlatforms(a.ImageName).String())
	return nil
}

// InOrder builds a list of artifacts in dependency order.
func InOrder(ctx context.Context, out io.Writer, tags tag.ImageTags, platforms platform.Resolver, artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int, store ArtifactStore) ([]graph.Artifact, error) {
	// `concurrency` specifies the max number of builds that can run at any one time. If concurrency is 0, then all builds can run in parallel.
	if concurrency == 0 {
		concurrency = len(artifacts)
	}
	if concurrency > 1 {
		output.Default.Fprintf(out, "Building %d artifacts in parallel\n", concurrency)
	}
	s := newScheduler(artifacts, artifactBuilder, concurrency, out, store)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return s.run(ctx, tags, platforms)
}

func performBuild(ctx context.Context, cw io.Writer, tags tag.ImageTags, platforms platform.Resolver, artifact *latest.Artifact, build ArtifactBuilder) (string, error) {
	tag, present := tags[artifact.ImageName]
	if !present {
		return "", fmt.Errorf("unable to find tag for image %s", artifact.ImageName)
	}
	pl := platforms.GetPlatforms(artifact.ImageName)
	if pl.IsNotEmpty() {
		output.Default.Fprintf(cw, "Target platforms: [%s]\n", pl)
	}
	tag = docker.SanitizeImageName(tag)
	return build(ctx, cw, artifact, tag, pl)
}
