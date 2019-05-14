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
	"github.com/pkg/errors"
)

const bufferedLinesPerArtifact = 10000

type artifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)

// InParallel builds a list of artifacts in parallel but prints the logs in sequential order.
func InParallel(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact artifactBuilder) ([]Artifact, error) {
	if len(artifacts) == 1 {
		return InSequence(ctx, out, tags, artifacts, buildArtifact)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := new(sync.Map)
	outputs := make([]chan []byte, len(artifacts))

	// Run builds in //
	for index := range artifacts {
		i := index
		lines := make(chan []byte, bufferedLinesPerArtifact)
		outputs[i] = lines

		r, w := io.Pipe()
		cw := setUpColorWriter(w, out)

		// Run build and write output/logs to piped writer and store build result in
		// sync.Map
		go runBuild(ctx, cw, artifacts[i], tags, results, buildArtifact)
		// Read build output/logs and write to buffered channel
		go readOutputAndWriteToChannel(r, lines)

	}

	// Print logs and collect results in order.
	return printAndCollectResults(out, artifacts, outputs, results)
}

func runBuild(ctx context.Context, cw io.WriteCloser, artifact *latest.Artifact, tags tag.ImageTags, results *sync.Map, build artifactBuilder) {
	color.Default.Fprintf(cw, "Building [%s]...\n", artifact.ImageName)

	event.BuildInProgress(artifact.ImageName)

	finalTag, err := getBuildResult(ctx, cw, artifact, tags, build)
	if err != nil {
		event.BuildFailed(artifact.ImageName, err)
		results.Store(artifact.ImageName, err)
	} else {
		results.Store(artifact.ImageName, finalTag)
	}

	event.BuildComplete(artifact.ImageName)
	cw.Close()
}

func readOutputAndWriteToChannel(r *io.PipeReader, lines chan []byte) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines <- scanner.Bytes()
	}
	close(lines)
}

func setUpColorWriter(w *io.PipeWriter, out io.Writer) io.WriteCloser {
	var cw io.WriteCloser
	if color.IsTerminal(out) {
		cw = color.ColoredWriteCloser{WriteCloser: w}
	} else {
		cw = w
	}
	return cw
}

func getBuildResult(ctx context.Context, cw io.WriteCloser, artifact *latest.Artifact, tags tag.ImageTags, build artifactBuilder) (string, error) {
	tag, present := tags[artifact.ImageName]
	if !present {
		return "", fmt.Errorf("unable to find tag for image %s", artifact.ImageName)
	}
	return build(ctx, cw, artifact, tag)
}

func printAndCollectResults(out io.Writer, artifacts []*latest.Artifact, outputs []chan []byte, results *sync.Map) ([]Artifact, error) {
	var built []Artifact

	for i, artifact := range artifacts {
		for line := range outputs[i] {
			out.Write(line)
			fmt.Fprintln(out)
		}

		if v, ok := results.Load(artifact.ImageName); ok {
			switch t := v.(type) {
			case error:
				return nil, errors.Wrapf(v.(error), "building [%s]", artifact.ImageName)
			case string:
				built = append(built, Artifact{ImageName: artifact.ImageName, Tag: v.(string)})
			default:
				return nil, fmt.Errorf("unknown type %T for %s", t, artifact.ImageName)
			}
		}
	}
	return built, nil
}
