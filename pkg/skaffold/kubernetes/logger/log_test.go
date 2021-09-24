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

package logger

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
			list := kubernetes.NewImageList()
			list.Add("image1")
			list.Add("image2")

			selected := list.Select(&v1.Pod{
				Spec: test.podSpec,
			})

			t.CheckDeepEqual(test.expectedMatch, selected)
		})
	}
}

func TestLogAggregatorZeroValue(t *testing.T) {
	var m *LogAggregator

	// Should not raise a nil dereference
	m.Start(context.Background(), ioutil.Discard)
	m.Mute()
	m.Unmute()
	m.Stop()
}

func TestPrefix(t *testing.T) {
	tests := []struct {
		description    string
		prefix         string
		pod            v1.Pod
		container      v1.ContainerStatus
		expectedPrefix string
	}{
		{
			description:    "auto (different names)",
			prefix:         "auto",
			pod:            podWithName("pod"),
			container:      containerWithName("container"),
			expectedPrefix: "[pod container]",
		},
		{
			description:    "auto (same names)",
			prefix:         "auto",
			pod:            podWithName("hello"),
			container:      containerWithName("hello"),
			expectedPrefix: "[hello]",
		},
		{
			description:    "container",
			prefix:         "container",
			pod:            podWithName("pod"),
			container:      containerWithName("container"),
			expectedPrefix: "[container]",
		},
		{
			description:    "podAndContainer (different names)",
			prefix:         "podAndContainer",
			pod:            podWithName("pod"),
			container:      containerWithName("container"),
			expectedPrefix: "[pod container]",
		},
		{
			description:    "podAndContainer (same names)",
			prefix:         "podAndContainer",
			pod:            podWithName("hello"),
			container:      containerWithName("hello"),
			expectedPrefix: "[hello hello]",
		},
		{
			description:    "none",
			prefix:         "none",
			pod:            podWithName("hello"),
			container:      containerWithName("hello"),
			expectedPrefix: "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			logger := NewLogAggregator(nil, nil, nil, &mockConfig{log: latestV2.LogsConfig{
				Prefix: test.prefix,
			}})

			p := logger.prefix(&test.pod, test.container)

			t.CheckDeepEqual(test.expectedPrefix, p)
		})
	}
}

func podWithName(n string) v1.Pod {
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: n,
		},
	}
}

func containerWithName(n string) v1.ContainerStatus {
	return v1.ContainerStatus{
		Name: n,
	}
}

type mockConfig struct {
	log latestV2.LogsConfig
}

func (c *mockConfig) Tail() bool {
	return true
}

func (c *mockConfig) PipelineForImage(string) (latestV2.Pipeline, bool) {
	var pipeline latestV2.Pipeline
	pipeline.Deploy.Logs = c.log
	return pipeline, true
}

func (c *mockConfig) DefaultPipeline() latestV2.Pipeline {
	var pipeline latestV2.Pipeline
	pipeline.Deploy.Logs = c.log
	return pipeline
}
