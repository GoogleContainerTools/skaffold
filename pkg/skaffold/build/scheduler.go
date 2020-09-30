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
	wg  sync.WaitGroup
}

func newScheduler(artifacts []*latest.Artifact, artifactBuilder ArtifactBuilder, concurrency int) *scheduler {
	s := scheduler{
		artifacts:       artifacts,
		artifactBuilder: artifactBuilder,
		nodes:           createNodes(artifacts),
		outputs:         make([]chan string, len(artifacts)),
		sem:             newCountingSemaphore(concurrency),
	}
	s.wg.Add(len(artifacts))
	for i := range artifacts {
		s.outputs[i] = make(chan string, buffSize)
	}
	return &s
}

func (s *scheduler) run(ctx context.Context, out io.Writer, tags tag.ImageTags) ([]Artifact, error) {
	for i := range s.artifacts {
		r, w := io.Pipe()

		// Create a goroutine for each element in dag. Each goroutine waits on its dependencies to finish building.
		// Because our artifacts form a DAG, at least one of the goroutines should be able to start building.
		go s.build(ctx, w, tags, i)

		// Read build output/logs and write to buffered channel
		go readOutputAndWriteToChannel(r, s.outputs[i])
	}
	s.wg.Wait()

	// Print logs and collect results in order.
	return collectResults(out, s.artifacts, &s.results, s.outputs)
}

func (s *scheduler) build(ctx context.Context, cw io.WriteCloser, tags tag.ImageTags, i int) {
	defer cw.Close()
	defer s.wg.Done()

	n := s.nodes[i]
	imageName := n.imageName

	err := n.waitForDependencies(ctx)
	release := s.sem.acquire()
	defer release()

	event.BuildInProgress(imageName)
	if err != nil {
		event.BuildFailed(imageName, err)
		s.results.Store(imageName, err)
		n.markFailure()
		return
	}

	finalTag, err := getBuildResult(ctx, cw, tags, s.artifacts[i], s.artifactBuilder)
	if err != nil {
		event.BuildFailed(imageName, err)
		s.results.Store(imageName, err)
		n.markFailure()
		return
	}

	event.BuildComplete(imageName)
	ar := Artifact{ImageName: imageName, Tag: finalTag}
	s.results.Store(ar.ImageName, ar)
	n.markSuccess()
}

func collectResults(out io.Writer, artifacts []*latest.Artifact, results *sync.Map, outputs []chan string) ([]Artifact, error) {
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
