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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/logfile"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type Muted interface {
	MuteBuild() bool
}

// WithLogFile wraps an `artifactBuilder` so that it optionally outputs its logs to a file.
func WithLogFile(builder ArtifactBuilder, muted Muted) ArtifactBuilder {
	if !muted.MuteBuild() {
		return builder
	}

	return func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
		file, err := logfile.Create("build", artifact.ImageName+".log")
		if err != nil {
			return "", fmt.Errorf("unable to create log file for %s: %w", artifact.ImageName, err)
		}
		fmt.Fprintln(out, " - writing logs to", file.Name())

		// Run the build.
		digest, err := builder(ctx, file, artifact, tag)

		// After the build finishes, close the log file.
		file.Close()

		return digest, err
	}
}
