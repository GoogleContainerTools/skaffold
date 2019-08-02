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

func TestNewPipelineResource(t *testing.T) {
	tests := []struct {
		description  string
		resourceName string
		resourceType tekton.PipelineResourceType
		params       []tekton.ResourceParam
		expected     *tekton.PipelineResource
	}{
		{
			description: "no params",
			expected: &tekton.PipelineResource{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PipelineResource",
					APIVersion: "tekton.dev/v1alpha1",
				},
			},
		},
		{
			description:  "normal params",
			resourceName: "test-resource",
			resourceType: tekton.PipelineResourceTypeGit,
			params: []tekton.ResourceParam{
				{
					Name:  "test-param",
					Value: "test-param-value",
				},
			},
			expected: &tekton.PipelineResource{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PipelineResource",
					APIVersion: "tekton.dev/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-resource",
				},
				Spec: tekton.PipelineResourceSpec{
					Type: "git",
					Params: []tekton.ResourceParam{
						{
							Name:  "test-param",
							Value: "test-param-value",
						},
					},
				},
			},
		},
		{
			description:  "empty params",
			resourceName: "",
			resourceType: "",
			params:       []tekton.ResourceParam{},
			expected: &tekton.PipelineResource{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PipelineResource",
					APIVersion: "tekton.dev/v1alpha1",
				},
				Spec: tekton.PipelineResourceSpec{
					Params: []tekton.ResourceParam{},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pipeline := NewPipelineResource(test.resourceName, test.resourceType, test.params)
			t.CheckDeepEqual(test.expected, pipeline)
		})
	}
}
