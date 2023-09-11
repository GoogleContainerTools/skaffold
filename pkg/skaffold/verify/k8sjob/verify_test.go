/*
Copyright 2023 The Skaffold Authors

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

package k8sjob

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPatchToK8sContainer(t *testing.T) {
	tests := []struct {
		description     string
		verifyContainer latest.VerifyContainer
		k8sContainer    corev1.Container
		expected        corev1.Container
	}{
		{
			description: "update all fields",
			verifyContainer: latest.VerifyContainer{
				Image:   "my-image:latest",
				Command: []string{"/bin/bash"},
				Args:    []string{"-c", "echo hello world"},
				Name:    "my-container",
				Env: []latest.VerifyEnvVar{
					{Name: "FOO", Value: "BAR"},
				},
			},
			k8sContainer: corev1.Container{},
			expected: corev1.Container{
				Image:   "my-image:latest",
				Command: []string{"/bin/bash"},
				Args:    []string{"-c", "echo hello world"},
				Name:    "my-container",
				Env: []corev1.EnvVar{
					{Name: "FOO", Value: "BAR"},
				},
			},
		},
		{
			description: "update image",
			verifyContainer: latest.VerifyContainer{
				Image: "my-new-image:latest",
			},
			k8sContainer: corev1.Container{
				Image: "my-image:latest",
			},
			expected: corev1.Container{
				Image: "my-new-image:latest",
			},
		},
		{
			description: "update command",
			verifyContainer: latest.VerifyContainer{
				Command: []string{"/bin/ls"},
			},
			k8sContainer: corev1.Container{
				Command: []string{"/bin/bash"},
			},
			expected: corev1.Container{
				Command: []string{"/bin/ls"},
			},
		},
		{
			description: "update args",
			verifyContainer: latest.VerifyContainer{
				Args: []string{"-l"},
			},
			k8sContainer: corev1.Container{
				Args: []string{"-c", "echo hello world"},
			},
			expected: corev1.Container{
				Args: []string{"-l"},
			},
		},
		{
			description: "update name",
			verifyContainer: latest.VerifyContainer{
				Name: "my-new-container",
			},
			k8sContainer: corev1.Container{
				Name: "my-container",
			},
			expected: corev1.Container{
				Name: "my-new-container",
			},
		},
		{
			description: "update env",
			verifyContainer: latest.VerifyContainer{
				Env: []latest.VerifyEnvVar{
					{Name: "FOO", Value: "BARR"},
					{Name: "BAZ", Value: "QUX"},
				},
			},
			k8sContainer: corev1.Container{
				Env: []corev1.EnvVar{
					{Name: "FOO", Value: "BAR"},
				},
			},
			// Duplicate name should be ok, as the last writer should win.
			expected: corev1.Container{
				Env: []corev1.EnvVar{
					{Name: "FOO", Value: "BAR"},
					{Name: "FOO", Value: "BARR"},
					{Name: "BAZ", Value: "QUX"},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			patchToK8sContainer(test.verifyContainer, &test.k8sContainer)
			t.CheckDeepEqual(test.expected, test.k8sContainer)
		})
	}
}
