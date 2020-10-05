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
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const bufferedLinesPerArtifact = 10000

type ArtifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)

// For testing
var (
	buffSize = bufferedLinesPerArtifact
)

type scheduler struct {
	artifacts       []*latest.Artifact
	artifactBuilder ArtifactBuilder

	nodes []buildNode // size len(artifacts)

	outputs []chan string // size len(artifacts)
	results sync.Map      // map[artifact.name]error

	sem countingSemaphore
}

func newScheduler(artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int) *scheduler {
	s := scheduler{
		artifacts:       artifacts,
		artifactBuilder: artifactBuilder,
		nodes:           createNodes(artifacts),
		outputs:         make([]chan string, len(artifacts)),
		sem:             newCountingSemaphore(concurrency),
	}
	for i := range artifacts {
		s.outputs[i] = make(chan string, buffSize)
	}
	return &s
}

func (s *scheduler) run(ctx context.Context, out io.Writer, tags tag.ImageTags) ([]Artifact, error) {
	g, gCtx := errgroup.WithContext(ctx)

	for i := range s.artifacts {
		i := i
		r, w := io.Pipe()

		// Create a goroutine for each element in dag. Each goroutine waits on its dependencies to finish building.
		// Because our artifacts form a DAG, at least one of the goroutines should be able to start building.
		// wrap in an error group so that all other builds are cancelled as soon as any one fails.
		g.Go(func() error {
			return s.build(gCtx, w, tags, i)
		})

		// Read build output/logs and write to buffered channel
		go readOutputAndWriteToChannel(r, s.outputs[i])
	}
	err := g.Wait()

	// Print logs and collect results in order.
	return collectResults(out, s.artifacts, &s.results, s.outputs, err)
}

func (s *scheduler) build(ctx context.Context, cw io.WriteCloser, tags tag.ImageTags, i int) error {
	defer cw.Close()

	n := s.nodes[i]
	imageName := n.imageName

	err := n.waitForDependencies(ctx)
	release := s.sem.acquire()
	defer release()

	event.BuildInProgress(imageName)
	if err != nil {
		event.BuildFailed(imageName, err)
		s.results.Store(imageName, err)
		return err
	}

	finalTag, err := getBuildResult(ctx, cw, tags, s.artifacts[i], s.artifactBuilder)
	if err != nil {
		event.BuildFailed(imageName, err)
		s.results.Store(imageName, err)
		return err
	}

	event.BuildComplete(imageName)
	ar := Artifact{ImageName: imageName, Tag: finalTag}
	s.results.Store(ar.ImageName, ar)
	n.markComplete()
	return nil
}

func collectResults(out io.Writer, artifacts []*latest.Artifact, results *sync.Map, outputs []chan string, cancelError error) ([]Artifact, error) {
	var built []Artifact
	for i, artifact := range artifacts {
		// Wait for build to complete.
		printResult(out, outputs[i])
		v, ok := results.Load(artifact.ImageName)
		if !ok {
			return nil, fmt.Errorf("could not find build result for image %s", artifact.ImageName)
		}
		switch t := v.(type) {
		case error:
			if t == context.Canceled {
				return nil, fmt.Errorf("build cancelled for %q due to another build failure: %w", artifact.ImageName, cancelError)
			}
			return nil, fmt.Errorf("couldn't build %q: %w", artifact.ImageName, t)
		case Artifact:
			built = append(built, t)
		default:
			return nil, fmt.Errorf("unknown type %T for %s", t, artifact.ImageName)
		}
	}
	return built, nil
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

func readOutputAndWriteToChannel(r io.Reader, lines chan string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	close(lines)
}

func getBuildResult(ctx context.Context, cw io.Writer, tags tag.ImageTags, artifact *latest.Artifact, build ArtifactBuilder) (string, error) {
	color.Default.Fprintf(cw, "Building [%s]...\n", artifact.ImageName)
	tag, present := tags[artifact.ImageName]
	if !present {
		return "", fmt.Errorf("unable to find tag for image %s", artifact.ImageName)
	}
	return build(ctx, cw, artifact, tag)
}

func printResult(out io.Writer, output chan string) {
	for line := range output {
		fmt.Fprintln(out, line)
	}
}
