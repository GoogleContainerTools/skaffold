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

package cluster

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/sirupsen/logrus"
)

// logLevel makes sure kaniko logs at least at Info level.
func logLevel() logrus.Level {
	level := logrus.GetLevel()
	if level < logrus.InfoLevel {
		return logrus.InfoLevel
	}
	return level
}

type countWriter struct {
	io.Writer
	written int
}

func (w *countWriter) Write(p []byte) (n int, err error) {
	written, err := w.Writer.Write(p)
	w.written = written
	return written, err
}

func (b *Builder) kubectlLogs(ctx context.Context, out io.Writer, ns, name string, follow bool) error {
	var args []string
	if ns != "" {
		args = append(args, "-n", ns)
	}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, name, "-c", constants.DefaultKanikoContainerName)

	return b.kubectlcli.Run(ctx, nil, out, "logs", args...)
}

func (b *Builder) streamLogs(ctx context.Context, out io.Writer, ns, name string) func() {
	var wg sync.WaitGroup
	wg.Add(1)

	var retry int32 = 1
	go func() {
		countOut := &countWriter{Writer: out}

		// Wait for logs to be available
		for atomic.LoadInt32(&retry) == 1 {
			if err := b.kubectlLogs(ctx, countOut, ns, name, true); err != nil {
				logrus.Debugln("unable to get kaniko pod logs:", err)
				time.Sleep(1 * time.Second)
				continue
			}

			break
		}

		// get latest logs if pod was terminated before logs have been streamed
		if countOut.written == 0 {
			b.kubectlLogs(ctx, out, ns, name, false)
		}

		wg.Done()
	}()

	return func() {
		atomic.StoreInt32(&retry, 0)
		wg.Wait()
	}
}
