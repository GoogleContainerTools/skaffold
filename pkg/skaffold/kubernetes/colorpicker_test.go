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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestColorPicker(t *testing.T) {
	var tests = []struct {
		description   string
		pod           *v1.Pod
		expectedColor color.Color
	}{
		{
			description: "not found",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "unknown",
				},
			},
			expectedColor: color.None,
		},
		{
			description: "found",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
				},
			},
			expectedColor: colorCodes[0],
		},
		{
			description: "second pod",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "second",
				},
			},
			expectedColor: colorCodes[1],
		},
	}

	picker := NewColorPicker()
	// register "second" twice and still expect colorCodes[1]
	for _, v := range []string{"pod", "second", "second"} {
		picker.Register(&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: v,
			},
		})
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			color := picker.Pick(test.pod)

			if color != test.expectedColor {
				t.Errorf("Expected color %d. Got %d", test.expectedColor, color)
			}
		})
	}
}
