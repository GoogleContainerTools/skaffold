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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewPipeline(t *testing.T) {
	tests := []struct {
		description  string
		pipelineName string
		resources    []tekton.PipelineDeclaredResource
		tasks        []tekton.PipelineTask
		expected     *tekton.Pipeline
	}{
		{
			description: "no params",
			expected: &tekton.Pipeline{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pipeline",
					APIVersion: "tekton.dev/v1alpha1",
				},
			},
		},
		{
			description:  "normal params",
			pipelineName: "pipeline-test",
			resources: []tekton.PipelineDeclaredResource{
				{
					Name: "test-resource1",
					Type: tekton.PipelineResourceTypeGit,
				},
				{
					Name: "test-resource2",
					Type: tekton.PipelineResourceTypeImage,
				},
			},
			tasks: []tekton.PipelineTask{
				{
					Name: "test-task1-pipeline",
					TaskRef: tekton.TaskRef{
						Name: "test-task1",
					},
				},
			},
			expected: &tekton.Pipeline{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pipeline",
					APIVersion: "tekton.dev/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "pipeline-test",
				},
				Spec: tekton.PipelineSpec{
					Resources: []tekton.PipelineDeclaredResource{
						{
							Name: "test-resource1", Type: "git",
						},
						{
							Name: "test-resource2", Type: "image",
						},
					},
					Tasks: []tekton.PipelineTask{
						{
							Name: "test-task1-pipeline",
							TaskRef: tekton.TaskRef{
								Name: "test-task1",
							},
						},
					},
				},
			},
		},
		{
			description:  "empty params",
			pipelineName: "",
			resources:    []tekton.PipelineDeclaredResource{},
			tasks:        []tekton.PipelineTask{},
			expected: &tekton.Pipeline{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pipeline",
					APIVersion: "tekton.dev/v1alpha1",
				},
				Spec: tekton.PipelineSpec{
					Resources: []tekton.PipelineDeclaredResource{},
					Tasks:     []tekton.PipelineTask{},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pipeline := NewPipeline(test.pipelineName, test.resources, test.tasks)
			t.CheckDeepEqual(test.expected, pipeline)
		})
	}
}
