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
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestColorPicker(t *testing.T) {
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
			expectedColor: colorCodes[0],
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
			expectedColor: colorCodes[0],
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
			expectedColor: colorCodes[1],
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
			expectedColor: colorCodes[1],
		},
	}

	picker := NewColorPicker()
	picker.AddImage("image:ignored")
	picker.AddImage("second")

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			color := picker.Pick(test.pod)

			t.CheckTrue(test.expectedColor == color)
		})
	}
}
