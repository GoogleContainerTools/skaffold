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

type artifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)

// For testing
var (
	buffSize      = bufferedLinesPerArtifact
	runInSequence = InSequence
)

// InParallel builds a list of artifacts in parallel but prints the logs in sequential order.
func InParallel(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact artifactBuilder, concurrency int) ([]Artifact, error) {
	if len(artifacts) == 0 {
		return nil, nil
	}

	if len(artifacts) == 1 || concurrency == 1 {
		return runInSequence(ctx, out, tags, artifacts, buildArtifact)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := new(sync.Map)
	outputs := make([]chan string, len(artifacts))

	if concurrency == 0 {
		concurrency = len(artifacts)
	}
	sem := make(chan bool, concurrency)

	// Run builds in //
	wg.Add(len(artifacts))
	for i := range artifacts {
		outputs[i] = make(chan string, buffSize)
		r, w := io.Pipe()

		// Run build and write output/logs to piped writer and store build result in
		// sync.Map
		go func(i int) {
			sem <- true
			runBuild(ctx, w, tags, artifacts[i], results, buildArtifact)
			<-sem

			wg.Done()
		}(i)

		// Read build output/logs and write to buffered channel
		go readOutputAndWriteToChannel(r, outputs[i])
	}

	// Print logs and collect results in order.
	return collectResults(out, artifacts, results, outputs)
}

func runBuild(ctx context.Context, cw io.WriteCloser, tags tag.ImageTags, artifact *latest.Artifact, results *sync.Map, build artifactBuilder) {
	event.BuildInProgress(artifact.ImageName)

	finalTag, err := getBuildResult(ctx, cw, tags, artifact, build)
	if err != nil {
		event.BuildFailed(artifact.ImageName, err)
		results.Store(artifact.ImageName, err)
	} else {
		event.BuildComplete(artifact.ImageName)
		artifact := Artifact{ImageName: artifact.ImageName, Tag: finalTag}
		results.Store(artifact.ImageName, artifact)
	}
	cw.Close()
}

func readOutputAndWriteToChannel(r io.Reader, lines chan string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	close(lines)
}

func getBuildResult(ctx context.Context, cw io.Writer, tags tag.ImageTags, artifact *latest.Artifact, build artifactBuilder) (string, error) {
	color.Default.Fprintf(cw, "Building [%s]...\n", artifact.ImageName)
	tag, present := tags[artifact.ImageName]
	if !present {
		return "", fmt.Errorf("unable to find tag for image %s", artifact.ImageName)
	}
	return build(ctx, cw, artifact, tag)
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

func printResult(out io.Writer, output chan string) {
	for line := range output {
		fmt.Fprintln(out, line)
	}
}
