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

package pipeline

import (
	"testing"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewTask(t *testing.T) {
	tests := []struct {
		description string
		taskName    string
		inputs      *tekton.Inputs
		steps       []v1.Container
		expected    *tekton.Task
	}{
		{
			description: "no params",
			expected: &tekton.Task{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Task",
					APIVersion: "tekton.dev/v1alpha1",
				},
				Spec: tekton.TaskSpec{},
			},
		},
		{
			description: "normal params",
			taskName:    "task-test",
			inputs: &tekton.Inputs{
				Resources: []tekton.TaskResource{
					{
						Name: "source",
						Type: tekton.PipelineResourceTypeGit,
					},
				},
			},
			steps: []v1.Container{
				{
					Name:    "step1",
					Image:   "test-image",
					Command: []string{"run", "test"},
					Args:    []string{"--test-arg"},
				},
			},
			expected: &tekton.Task{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Task",
					APIVersion: "tekton.dev/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "task-test",
				},
				Spec: tekton.TaskSpec{
					Inputs: &tekton.Inputs{
						Resources: []tekton.TaskResource{
							{
								Name: "source",
								Type: "git",
							},
						},
					},
					Steps: []v1.Container{
						{
							Name:    "step1",
							Image:   "test-image",
							Command: []string{"run", "test"},
							Args:    []string{"--test-arg"},
						},
					},
				},
			},
		},
		{
			description: "empty params",
			taskName:    "",
			inputs:      &tekton.Inputs{},
			steps:       []v1.Container{},
			expected: &tekton.Task{
				TypeMeta: metav1.TypeMeta{Kind: "Task", APIVersion: "tekton.dev/v1alpha1"},
				Spec: tekton.TaskSpec{
					Inputs: &tekton.Inputs{},
					Steps:  []v1.Container{},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pipeline := NewTask(test.taskName, test.inputs, nil, test.steps, nil)
			t.CheckDeepEqual(test.expected, pipeline)
		})
	}
}
