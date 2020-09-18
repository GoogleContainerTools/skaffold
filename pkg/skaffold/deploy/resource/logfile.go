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

package resource

import (
	"bytes"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/logfile"
)

// withLogFile returns a multiwriter that writes both to a file and a buffer, with the buffer being written to the provided output buffer in case of error
func withLogFile(container string, out io.Writer, l []string, muted bool) (io.Writer, func([]string), error) {
	if !muted || len(l) <= maxLogLines {
		return out, func([]string) {}, nil
	}
	file, err := logfile.Create("statuscheck", container+".log")
	if err != nil {
		return out, func([]string) {}, fmt.Errorf("unable to create log file for statuscheck step: %w", err)
	}

	// Print logs to a memory buffer and to a file.
	var buf bytes.Buffer
	w := io.MultiWriter(file, &buf)

	// After the status check updates finishes, close the log file.
	return w, func(lines []string) {
		file.Close()
		// Write last few lines to out
		for _, l := range lines {
			out.Write([]byte(l))
		}
		fmt.Fprintf(out, "%s %s Full logs at %s\n", tab, tab, file.Name())
	}, err
}
