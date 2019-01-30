/*
Copyright 2018 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

const bufferedLinesPerArtifact = 10000

type artifactBuilder func(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error)

// InParallel builds a list of artifacts in parallel but prints the logs in sequential order.
func InParallel(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact, buildArtifact artifactBuilder) ([]Artifact, error) {
	if len(artifacts) == 1 {
		return InSequence(ctx, out, tagger, artifacts, buildArtifact)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	n := len(artifacts)
	tags := make([]string, n)
	errs := make([]error, n)
	outputs := make([]chan []byte, n)

	// Run builds in //
	for index := range artifacts {
		i := index
		lines := make(chan []byte, bufferedLinesPerArtifact)
		outputs[i] = lines

		r, w := io.Pipe()

		// Log to the pipe, output will be collected and printed later
		go func() {
			// Make sure logs are printed in colors
			var cw io.WriteCloser
			if color.IsTerminal(out) {
				cw = color.ColoredWriteCloser{WriteCloser: w}
			} else {
				cw = w
			}

			color.Default.Fprintf(cw, "Building [%s]...\n", artifacts[i].ImageName)
			tags[i], errs[i] = buildArtifact(ctx, cw, tagger, artifacts[i])
			cw.Close()
		}()

		go func() {
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				lines <- scanner.Bytes()
			}
			close(lines)
		}()
	}

	// Print logs and collect results in order.
	var built []Artifact

	for i, artifact := range artifacts {
		for line := range outputs[i] {
			out.Write(line)
			fmt.Fprintln(out)
		}

		if errs[i] != nil {
			return nil, errors.Wrapf(errs[i], "building [%s]", artifact.ImageName)
		}

		built = append(built, Artifact{
			ImageName: artifact.ImageName,
			Tag:       tags[i],
		})
	}

	return built, nil
}
