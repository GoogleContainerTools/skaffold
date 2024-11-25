package k8sjob

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	corev1 "k8s.io/api/core/v1"
)

func TestPatchToK8sContainer(t *testing.T) {
	tests := []struct {
		description     string
		actionContainer corev1.Container
		k8sContainer    corev1.Container
		expected        corev1.Container
	}{
		{
			description: "update all fields",
			actionContainer: corev1.Container{
				Image:   "my-image:latest",
				Command: []string{"/bin/bash"},
				Args:    []string{"-c", "echo hello world"},
				Name:    "my-container",
				Env: []corev1.EnvVar{
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
			actionContainer: corev1.Container{
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
			actionContainer: corev1.Container{
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
			actionContainer: corev1.Container{
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
			actionContainer: corev1.Container{
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
			actionContainer: corev1.Container{
				Env: []corev1.EnvVar{
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
			patchToK8sContainer(test.actionContainer, &test.k8sContainer)
			t.CheckDeepEqual(test.expected, test.k8sContainer)
		})
	}
}
