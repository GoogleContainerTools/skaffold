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

type ArtifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)

// For testing
var (
	buffSize = bufferedLinesPerArtifact
)

// Create builds a list of artifacts in dependency order within the specified max concurrency
func Create(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact ArtifactBuilder, concurrency int) ([]Artifact, error) {
	m := new(sync.Map)
	for _, a := range artifacts {
		m.Store(a.ImageName, make(chan interface{}))
	}

	var awdSlice []artifactWithDeps
	for _, a := range artifacts {
		awd := artifactWithDeps{Artifact: a}
		for _, d := range a.Dependencies {
			ch, _ := m.Load(d.ImageName)
			awd.Deps = append(awd.Deps, ch.(chan interface{}))
		}
		awdSlice = append(awdSlice, awd)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := new(sync.Map)
	outputs := make([]chan string, len(awdSlice))

	if concurrency == 0 {
		concurrency = len(awdSlice)
	}
	sem := make(chan bool, concurrency)

	wg.Add(len(artifacts))
	for i := range awdSlice {
		outputs[i] = make(chan string, buffSize)
		r, w := io.Pipe()

		// Run build and write output/logs to piped writer and store build result in
		// sync.Map
		go func(i int) {
			for _, dep := range awdSlice[i].Deps {
				// wait for dependency to complete build
				<-dep
			}
			sem <- true
			runBuild(ctx, w, tags, awdSlice[i].Artifact, results, buildArtifact)
			ch, _ := m.Load(awdSlice[i].Artifact.ImageName)
			// closing channel notifies all listeners waiting for this build to complete
			close(ch.(chan interface{}))
			<-sem

			wg.Done()
		}(i)

		// Read build output/logs and write to buffered channel
		go readOutputAndWriteToChannel(r, outputs[i])
	}

	// Print logs and collect results in order.
	return collectResults(out, artifacts, results, outputs)
}

type artifactWithDeps struct {
	Artifact *latest.Artifact
	Deps     []chan interface{}
}

func runBuild(ctx context.Context, cw io.WriteCloser, tags tag.ImageTags, artifact *latest.Artifact, results *sync.Map, build ArtifactBuilder) {
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

func getBuildResult(ctx context.Context, cw io.Writer, tags tag.ImageTags, artifact *latest.Artifact, build ArtifactBuilder) (string, error) {
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
