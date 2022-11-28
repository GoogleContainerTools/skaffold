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
	"sync"

	"github.com/sirupsen/logrus"

	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
)

//nolint:golint
func StreamRequest(ctx context.Context, out io.Writer, headerColor output.Color, prefix, podName, containerName string, stopper chan bool, lock sync.Locker, isMuted func() bool, rc io.Reader) error {
	r := bufio.NewReader(rc)
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("%s interrupted", prefix)
			return nil
		case <-stopper:
			return nil
		default:
			// Read up to newline
			line, err := r.ReadString('\n')
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("reading bytes from log stream: %w", err)
			}

			formattedLine := headerColor.Sprintf("%s ", prefix) + line
			printLogLine(headerColor, out, isMuted, lock, prefix, line)
			eventV2.ApplicationLog(podName, containerName, line, formattedLine)
		}
	}
}

func printLogLine(headerColor output.Color, out io.Writer, isMuted func() bool, lock sync.Locker, prefix, text string) {
	if !isMuted() {
		lock.Lock()

		headerColor.Fprintf(out, "%s ", prefix)
		fmt.Fprint(out, text)

		lock.Unlock()
	}
}
