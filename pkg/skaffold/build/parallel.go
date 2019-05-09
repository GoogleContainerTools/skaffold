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

// InParallel builds a list of artifacts in parallel but prints the logs in sequential order.
func InParallel(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact artifactBuilder) (<-chan Result, error) {
	if len(artifacts) == 1 {
		return InSequence(ctx, out, tags, artifacts, buildArtifact)
	}

	resultChan := make(chan Result, len(artifacts))

	ctx, cancel := context.WithCancel(ctx)

	// for collecting all output and printing in order later on
	outputs := make([]chan []byte, len(artifacts))

	allBuildsWg := &sync.WaitGroup{}

	for i, a := range artifacts {
		// give each build a byte buffer to write its output to
		lines := make(chan []byte, bufferedLinesPerArtifact)
		outputs[i] = lines

		//	allBuildsWg.Add(1)
		go func(artifact *latest.Artifact, c chan Result, lines chan []byte) {
			defer allBuildsWg.Done()
			res := &Result{
				Target: *artifact,
			}
			wg := &sync.WaitGroup{}

			r, w := io.Pipe()

			// Log to the pipe, output will be collected and printed later
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Make sure logs are printed in colors
				var cw io.WriteCloser
				if color.IsTerminal(out) {
					cw = color.ColoredWriteCloser{WriteCloser: w}
				} else {
					cw = w
				}

				color.Default.Fprintf(cw, "Building [%s]...\n", artifact.ImageName)

				event.BuildInProgress(artifact.ImageName)

				tag, present := tags[artifact.ImageName]
				if !present {
					res.Error = fmt.Errorf("building [%s]: unable to find tag for image", artifact.ImageName)
					event.BuildFailed(artifact.ImageName, res.Error)
				} else {
					bRes, err := buildArtifact(ctx, cw, artifact, tag)
					if err != nil {
						res.Error = err
						event.BuildFailed(artifact.ImageName, err)
					} else {
						res.Result = Artifact{
							ImageName: artifact.ImageName,
							Tag:       bRes,
						}
					}
				}
				event.BuildComplete(artifact.ImageName)
				cw.Close()
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(r)
				for scanner.Scan() {
					lines <- scanner.Bytes()
				}
				close(lines)
			}()

			wg.Wait() // wait for build to finish and output to be processed
			c <- *res // send the result back through the results channel
		}(a, resultChan, lines)
	}

	go func() {
		defer cancel()
		for i := range artifacts {
			for line := range outputs[i] {
				out.Write(line)
				fmt.Fprintln(out)
			}
		}
		allBuildsWg.Wait()
		close(resultChan)
	}()
	return resultChan, nil
}
