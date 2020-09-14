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

// artifactChanModel models the artifact dependency graph using a set of channels.
// Each artifact has a channel that it closes once it completes building (either success or failure) by calling `markComplete`. This notifies *all* listeners waiting for this artifact.
// Additionally it has a list of channels for each of its dependencies.
// Calling `waitForDependencies` ensures that all required artifacts' channels have already been closed and as such have finished building.
// This model allows for composing any arbitrary function with dependency ordering.
type artifactChanModel struct {
	artifact              *latest.Artifact
	artifactChan          chan interface{}
	requiredArtifactChans []chan interface{}
}

func (a artifactChanModel) markComplete() {
	// closing channel notifies all listeners waiting for this build to complete
	close(a.artifactChan)
}
func (a artifactChanModel) waitForDependencies(ctx context.Context) {
	for _, dep := range a.requiredArtifactChans {
		// wait for dependency to complete build
		select {
		case <-ctx.Done():
		case <-dep:
		}
	}
}

// InOrder builds a list of artifacts in dependency order.
// `concurrency` specifies the max number of builds that can run at any one time. If concurrency is 0, then it's set to the length of the `artifacts` slice.
func InOrder(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact ArtifactBuilder, concurrency int) ([]Artifact, error) {
	acmSlice := makeArtifactChanModel(artifacts)
	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := new(sync.Map)
	outputs := make([]chan string, len(acmSlice))

	if concurrency == 0 {
		concurrency = len(acmSlice)
	}
	sem := make(chan bool, concurrency)

	wg.Add(len(artifacts))
	for i := range acmSlice {
		outputs[i] = make(chan string, buffSize)
		r, w := io.Pipe()

		// Create a goroutine for each element in acmSlice. Each goroutine waits on its dependencies to finish building.
		// Because our artifacts form a DAG, at least one of the goroutines should be able to start building.
		go func(i int) {
			acmSlice[i].waitForDependencies(ctx)
			sem <- true
			// Run build and write output/logs to piped writer and store build result in sync.Map
			runBuild(ctx, w, tags, acmSlice[i].artifact, results, buildArtifact)
			acmSlice[i].markComplete()
			<-sem

			wg.Done()
		}(i)

		// Read build output/logs and write to buffered channel
		go readOutputAndWriteToChannel(r, outputs[i])
	}

	// Print logs and collect results in order.
	return collectResults(out, artifacts, results, outputs)
}

func makeArtifactChanModel(artifacts []*latest.Artifact) []artifactChanModel {
	chanMap := make(map[string]chan interface{})
	for _, a := range artifacts {
		chanMap[a.ImageName] = make(chan interface{})
	}

	var acmSlice []artifactChanModel
	for _, a := range artifacts {
		acm := artifactChanModel{artifact: a, artifactChan: chanMap[a.ImageName]}
		for _, d := range a.Dependencies {
			acm.requiredArtifactChans = append(acm.requiredArtifactChans, chanMap[d.ImageName])
		}
		acmSlice = append(acmSlice, acm)
	}
	return acmSlice
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
