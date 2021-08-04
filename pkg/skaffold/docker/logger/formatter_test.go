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

package logger

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/tracker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type mockColorPicker struct{}

func (m *mockColorPicker) Pick(image string) output.Color {
	return output.Default
}

func (m *mockColorPicker) AddImage(string) {}

func TestPrintLogLine(t *testing.T) {
	testutil.Run(t, "verify lines are not intermixed", func(t *testutil.T) {
		var (
			buf bytes.Buffer
			wg  sync.WaitGroup

			linesPerGroup = 100
			groups        = 5
		)

		f := NewDockerLogFormatter(&mockColorPicker{}, tracker.NewContainerTracker(), func() bool { return false }, "id")

		for i := 0; i < groups; i++ {
			wg.Add(1)

			go func() {
				for i := 0; i < linesPerGroup; i++ {
					f.PrintLine(&buf, "TEXT\n")
				}
				wg.Done()
			}()
		}
		wg.Wait()

		lines := strings.Split(buf.String(), "\n")
		for i := 0; i < groups*linesPerGroup; i++ {
			t.CheckDeepEqual("TEXT", lines[i])
		}
	})
}

func TestPrintLogLineFormatted(t *testing.T) {
	testutil.Run(t, "verify lines have correct prefix", func(t *testutil.T) {
		var (
			buf bytes.Buffer
			wg  sync.WaitGroup

			linesPerGroup = 100
			groups        = 5
		)
		ct := tracker.NewContainerTracker()
		ct.Add(graph.Artifact{ImageName: "image", Tag: "image:tag"}, tracker.Container{ID: "id"})

		f := NewDockerLogFormatter(&mockColorPicker{}, ct, func() bool { return false }, "id")

		for i := 0; i < groups; i++ {
			wg.Add(1)

			go func() {
				for i := 0; i < linesPerGroup; i++ {
					f.PrintLine(&buf, "TEXT\n")
				}
				wg.Done()
			}()
		}
		wg.Wait()

		lines := strings.Split(buf.String(), "\n")
		for i := 0; i < groups*linesPerGroup; i++ {
			t.CheckDeepEqual("[image] TEXT", lines[i])
		}
	})
}
