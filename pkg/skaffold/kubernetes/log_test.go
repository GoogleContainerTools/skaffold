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

package kubernetes

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
)

func TestSinceSeconds(t *testing.T) {
	tests := []struct {
		description string
		duration    time.Duration
		expected    int64
	}{
		{"0s", 0, 1},
		{"1ms", 1 * time.Millisecond, 1},
		{"500ms", 500 * time.Millisecond, 1},
		{"999ms", 999 * time.Millisecond, 1},
		{"1s", 1 * time.Second, 1},
		{"1.1s", 1100 * time.Millisecond, 2},
		{"1.5s", 1500 * time.Millisecond, 2},
		{"1.9s", 1500 * time.Millisecond, 2},
		{"2s", 2 * time.Second, 2},
		{"10s", 10 * time.Second, 10},
		{"60s", 60 * time.Second, 60},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			since := sinceSeconds(test.duration)

			t.CheckDeepEqual(test.expected, since)
		})
	}
}

func TestSelect(t *testing.T) {
	tests := []struct {
		description   string
		podSpec       v1.PodSpec
		expectedMatch bool
	}{
		{
			description:   "match container",
			podSpec:       v1.PodSpec{Containers: []v1.Container{{Image: "image1"}}},
			expectedMatch: true,
		},
		{
			description:   "match init container",
			podSpec:       v1.PodSpec{InitContainers: []v1.Container{{Image: "image2"}}},
			expectedMatch: true,
		},
		{
			description:   "no match",
			podSpec:       v1.PodSpec{Containers: []v1.Container{{Image: "image3"}}},
			expectedMatch: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			list := NewImageList()
			list.Add("image1")
			list.Add("image2")

			selected := list.Select(&v1.Pod{
				Spec: test.podSpec,
			})

			t.CheckDeepEqual(test.expectedMatch, selected)
		})
	}
}

func TestPrintLogLine(t *testing.T) {
	testutil.Run(t, "verify lines are not intermixed", func(t *testutil.T) {
		var buf bytes.Buffer

		logger := &LogAggregator{
			output: &buf,
		}

		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)

			go func() {
				for i := 0; i < 100; i++ {
					logger.printLogLine(color.Default, "PREFIX", "TEXT\n")
				}
				wg.Done()
			}()
		}
		wg.Wait()

		lines := strings.Split(buf.String(), "\n")
		for i := 0; i < 5*100; i++ {
			t.CheckDeepEqual("PREFIX TEXT", lines[i])
		}
	})
}
