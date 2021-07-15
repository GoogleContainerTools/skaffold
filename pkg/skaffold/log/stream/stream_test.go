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
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintLogLine(t *testing.T) {
	testutil.Run(t, "verify lines are not intermixed", func(t *testutil.T) {
		var (
			buf  bytes.Buffer
			wg   sync.WaitGroup
			lock sync.Mutex

			linesPerGroup = 100
			groups        = 5
		)

		for i := 0; i < groups; i++ {
			wg.Add(1)

			go func() {
				for i := 0; i < linesPerGroup; i++ {
					printLogLine(output.Default, &buf, func() bool { return false }, &lock, "PODNAME", "CONTAINERNAME", "PREFIX", "TEXT\n")
				}
				wg.Done()
			}()
		}
		wg.Wait()

		lines := strings.Split(buf.String(), "\n")
		for i := 0; i < groups*linesPerGroup; i++ {
			t.CheckDeepEqual("PREFIX TEXT", lines[i])
		}
	})
}
