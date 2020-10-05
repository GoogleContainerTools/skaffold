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
	artifactBuilder ArtifactBuilder
	nodes           []buildNode // size len(artifacts)
	logger          logWriter
	results         resultStore
	sem             countingSemaphore
}

func newScheduler(artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int) *scheduler {
	s := scheduler{
		artifacts:       artifacts,
		artifactBuilder: artifactBuilder,
		nodes:           createNodes(artifacts),
		sem:             newCountingSemaphore(concurrency),
		results:         newResultStore(),
		logger:          newLogWriter(len(artifacts)),
	}
	return &s
}

func (s *scheduler) run(ctx context.Context, out io.Writer, tags tag.ImageTags) ([]Artifact, error) {
	g, gCtx := errgroup.WithContext(ctx)

	for i := range s.artifacts {
		i := i

		// Create a goroutine for each element in dag. Each goroutine waits on its dependencies to finish building.
		// Because our artifacts form a DAG, at least one of the goroutines should be able to start building.
		// wrap in an error group so that all other builds are cancelled as soon as any one fails.
		g.Go(func() error {
			return s.build(gCtx, tags, i)
		})
	}
	// print output for all artifact builds in order
	s.logger.PrintInOrder(out)
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return formatResults(s.artifacts, s.results)
}

// formatResults returns the build results in the order of `latest.Artifacts` in the input slice.
func formatResults(artifacts []*latest.Artifact, results resultStore) ([]Artifact, error) {
	var builds []Artifact
	for _, a := range artifacts {
		t, err := results.GetTag(a)
		if err != nil {
			return nil, err
		}
		builds = append(builds, Artifact{ImageName: a.ImageName, Tag: t})
	}
	return builds, nil
}

func (s *scheduler) build(ctx context.Context, tags tag.ImageTags, i int) error {
	n := s.nodes[i]
	a := s.artifacts[i]

	err := n.waitForDependencies(ctx)
	release := s.sem.acquire()
	defer release()
	w, _ := s.logger.GetWriter()
	defer w.Close()

	if err == context.Canceled {
		event.BuildCanceled(a.ImageName)
		return err
	}

	event.BuildInProgress(a.ImageName)
	if err != nil {
		event.BuildFailed(a.ImageName, err)
		return err
	}

	finalTag, err := getBuildResult(ctx, w, tags, s.artifacts[i], s.artifactBuilder)
	if err != nil {
		event.BuildFailed(a.ImageName, err)
		return err
	}

	event.BuildComplete(a.ImageName)
	s.results.Record(a, finalTag)
	n.markComplete()
	return nil
}

// InOrder builds a list of artifacts in dependency order.
func InOrder(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int) ([]Artifact, error) {
	// `concurrency` specifies the max number of builds that can run at any one time. If concurrency is 0, then all builds can run in parallel.
	if concurrency == 0 {
		concurrency = len(artifacts)
	}

	s := newScheduler(artifacts, artifactBuilder, concurrency)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return s.run(ctx, out, tags)
}

func getBuildResult(ctx context.Context, cw io.Writer, tags tag.ImageTags, artifact *latest.Artifact, build ArtifactBuilder) (string, error) {
	color.Default.Fprintf(cw, "Building [%s]...\n", artifact.ImageName)
	tag, present := tags[artifact.ImageName]
	if !present {
		return "", fmt.Errorf("unable to find tag for image %s", artifact.ImageName)
	}
	return build(ctx, cw, artifact, tag)
}
