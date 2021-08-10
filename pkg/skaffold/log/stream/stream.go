/*
Copyright 2021 The Skaffold Authors

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

package stream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
)

//nolint:golint
func StreamRequest(ctx context.Context, out io.Writer, formatter log.Formatter, rc io.Reader) error {
	r := bufio.NewReader(rc)
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("%s interrupted", formatter.Name())
			return nil
		default:
			// Read up to newline
			line, err := r.ReadString('\n')
			// As per https://github.com/kubernetes/kubernetes/blob/017b359770e333eacd3efcb4174f1d464c208400/test/e2e/storage/podlogs/podlogs.go#L214
			// Filter out the expected "end of stream" error message and
			// attempts to read logs from a container that isn't ready (yet?!).
			if err == io.EOF {
				if !isEmptyOrContainerNotReady(line) {
					formatter.PrintLine(out, line)
				}
				return nil
			}
			if err != nil {
				return fmt.Errorf("reading bytes from log stream: %w", err)
			}
			formatter.PrintLine(out, line)
		}
	}
}

func isEmptyOrContainerNotReady(line string) bool {
	return line == "" ||
		strings.HasPrefix(line, "rpc error: code = Unknown desc = Error: No such container:") ||
		strings.HasPrefix(line, "unable to retrieve container logs for ") ||
		strings.HasPrefix(line, "Unable to retrieve container logs for ")
}
