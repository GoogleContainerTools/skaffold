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

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type mockColorPicker struct{}

func (m *mockColorPicker) Pick(image string) output.Color {
	return output.Default
}

func (m *mockColorPicker) AddImage(string) {}

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

func (c *mockConfig) JSONParseConfig() latestV2.JSONParseConfig {
	return c.log.JSONParse
}

func TestPrintLogLine(t *testing.T) {
	testutil.Run(t, "verify lines are not intermixed", func(t *testutil.T) {
		var (
			buf bytes.Buffer
			wg  sync.WaitGroup

			linesPerGroup = 100
			groups        = 5
		)

		f := newKubernetesLogFormatter(&mockConfig{log: latestV2.LogsConfig{Prefix: "none"}}, &mockColorPicker{}, func() bool { return false }, &v1.Pod{}, v1.ContainerStatus{})

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

func TestColorForPod(t *testing.T) {
	tests := []struct {
		description   string
		pod           *v1.Pod
		expectedColor output.Color
	}{
		{
			description:   "not found",
			pod:           &v1.Pod{},
			expectedColor: output.None,
		},
		{
			description: "found",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{Image: "image"},
					},
				},
			},
			expectedColor: output.DefaultColorCodes[0],
		},
		{
			description: "ignore tag",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{Image: "image:tag"},
					},
				},
			},
			expectedColor: output.DefaultColorCodes[0],
		},
		{
			description: "second image",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{Image: "second:tag"},
					},
				},
			},
			expectedColor: output.DefaultColorCodes[1],
		},
		{
			description: "accept image with digest",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{Image: "second:tag@sha256:d3f4dd1541ee34b96850efc46955bada1a415b0594dc9948607c0197d2d16749"},
					},
				},
			},
			expectedColor: output.DefaultColorCodes[1],
		},
	}

	// artifacts are registered using their tag, since these have default repo substitutions applied
	p := output.NewColorPicker()
	p.AddImage("image")
	p.AddImage("second")

	for _, test := range tests {
		f := newKubernetesLogFormatter(&mockConfig{log: latestV2.LogsConfig{Prefix: "none"}}, p, func() bool { return false }, test.pod, v1.ContainerStatus{})

		testutil.Run(t, test.description, func(t *testutil.T) {
			color := f.color()

			t.CheckTrue(test.expectedColor == color)
		})
	}
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
			f := newKubernetesLogFormatter(&mockConfig{log: latestV2.LogsConfig{
				Prefix: test.prefix,
			}}, &mockColorPicker{}, func() bool { return false }, &test.pod, test.container)

			t.CheckDeepEqual(test.expectedPrefix, f.prefix)
		})
	}
}

func TestPrintline(t *testing.T) {
	tests := []struct {
		description string
		isMuted     bool
		expected    string
	}{
		{
			description: "muted",
			isMuted:     true,
		},
		{
			description: "unmuted",
			expected:    "[hello container]test line",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pod := podWithName("hello")
			f := newKubernetesLogFormatter(&mockConfig{log: latestV2.LogsConfig{
				Prefix: "auto",
			}}, &mockColorPicker{}, func() bool { return test.isMuted }, &pod,
				containerWithName("container"))
			var out bytes.Buffer
			f.PrintLine(&out, "test line")
			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}
