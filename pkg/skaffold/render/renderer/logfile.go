/*
Copyright 2022 The Skaffold Authors

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

package renderer

import (
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/logfile"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
)

type Muted interface {
	MuteRender() bool
}

// WithLogFile returns a file to write the deploy output to, and a function to be executed after the deploy step is complete.
func WithLogFile(filename string, out io.Writer, muted Muted) (io.Writer, func(), error) {
	if !muted.MuteRender() {
		return out, func() {}, nil
	}

	file, err := logfile.Create("render", filename)
	if err != nil {
		return out, func() {}, fmt.Errorf("unable to create log file for render step: %w", err)
	}

	output.Default.Fprintln(out, "Starting render...")
	fmt.Fprintln(out, "- writing logs to", file.Name())

	// After the render finishes, close the log file.
	return file, func() {
		file.Close()
	}, err
}
