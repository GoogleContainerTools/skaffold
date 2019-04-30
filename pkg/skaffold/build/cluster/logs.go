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
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// logLevel makes sure kaniko logs at least at Info level.
func logLevel() logrus.Level {
	level := logrus.GetLevel()
	if level < logrus.InfoLevel {
		return logrus.InfoLevel
	}
	return level
}

func streamLogs(out io.Writer, name string, pods corev1.PodInterface) func() {
	var wg sync.WaitGroup
	wg.Add(1)

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

			io.Copy(out, r)
			return
		}
	}()

	return func() {
		atomic.StoreInt32(&retry, 0)
		wg.Wait()
	}
}
