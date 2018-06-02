/*
Copyright 2018 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	v1 "k8s.io/api/core/v1"
)

func TestColorSprint(t *testing.T) {
	colored := colorCodeWhite.Sprint("TEXT")

	expected := "\033[97mTEXT\033[0m"
	if colored != expected {
		t.Errorf("Expected %s. Got %s", expected, colored)
	}
}

func TestColorPicker(t *testing.T) {
	var tests = []struct {
		description   string
		pod           *v1.Pod
		expectedColor color
	}{
		{
			description:   "not found",
			pod:           &v1.Pod{},
			expectedColor: colorCodeWhite,
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
	}

	picker := NewColorPicker([]*v1alpha2.Artifact{
		{ImageName: "image"},
		{ImageName: "second"},
	})

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			color := picker.Pick(test.pod)

			if color != test.expectedColor {
				t.Errorf("Expected color %d. Got %d", test.expectedColor, color)
			}
		})
	}
}
