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
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

// logLevel makes sure kaniko logs at least at Info level and at most Debug level (trace doesn't work with Kaniko)
func logLevel() logrus.Level {
	level := logrus.GetLevel()
	if level < logrus.InfoLevel {
		return logrus.InfoLevel
	}
	if level > logrus.DebugLevel {
		return logrus.DebugLevel
	}
	return level
}

func streamLogs(ctx context.Context, out io.Writer, name string, pods corev1.PodInterface) func() {
	var wg sync.WaitGroup
	wg.Add(1)

	var written int64
	var retry int32 = 1
	go func() {
		defer wg.Done()

		for atomic.LoadInt32(&retry) == 1 {
			r, err := pods.GetLogs(name, &v1.PodLogOptions{
				Follow:    true,
				Container: constants.DefaultKanikoContainerName,
			}).Stream()
			if err != nil {
				logrus.Debugln("unable to get kaniko pod logs:", err)
				time.Sleep(1 * time.Second)
				continue
			}

			scanner := bufio.NewScanner(r)
			for {
				select {
				case <-ctx.Done():
					return // The build was cancelled
				default:
					if !scanner.Scan() {
						return // No more logs
					}

					fmt.Fprintln(out, scanner.Text())
					atomic.AddInt64(&written, 1)
				}
			}
		}
	}()

	return func() {
		atomic.StoreInt32(&retry, 0)
		wg.Wait()

		// get latest logs if pod was terminated before logs have been streamed
		if atomic.LoadInt64(&written) == 0 {
			r, err := pods.GetLogs(name, &v1.PodLogOptions{
				Container: constants.DefaultKanikoContainerName,
			}).Stream()
			if err == nil {
				io.Copy(out, r)
			}
		}
	}
}
