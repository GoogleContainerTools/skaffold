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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
)

const bufferedLinesPerArtifact = 10000

type artifactBuilder func(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *v1alpha2.Artifact) (string, error)

// InParallel builds a list of artifacts in parallel but prints the logs in sequential order.
func InParallel(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact, buildArtifact artifactBuilder) ([]Artifact, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	n := len(artifacts)
	tags := make([]string, n)
	errs := make([]error, n)
	outputs := make([]chan (string), n)

	// Run builds in //
	for index := range artifacts {
		i := index
		lines := make(chan (string), bufferedLinesPerArtifact)
		outputs[i] = lines

		r, w := io.Pipe()

		go func() {
			fmt.Fprintf(w, "Building [%s]...\n", artifacts[i].ImageName)

			tags[i], errs[i] = buildArtifact(ctx, w, tagger, artifacts[i])
			w.Close()
		}()

		go func() {
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				lines <- scanner.Text()
			}
			close(lines)
		}()
	}

	// Print logs and collect results in order.
	var built []Artifact

	for i, artifact := range artifacts {
		for line := range outputs[i] {
			fmt.Fprintln(out, line)
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
