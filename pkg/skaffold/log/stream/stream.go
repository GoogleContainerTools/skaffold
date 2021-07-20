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
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("reading bytes from log stream: %w", err)
			}

			formatter.PrintLine(out, line)
		}
	}
}
