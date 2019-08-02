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
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewGitResource(resourceName, url string) *tekton.PipelineResource {
	params := []tekton.ResourceParam{
		{
			Name:  "url",
			Value: url,
		},
	}
	return NewPipelineResource(resourceName, tekton.PipelineResourceTypeGit, params)
}

func NewPipelineResource(resourceName string, resourceType tekton.PipelineResourceType, params []tekton.ResourceParam) *tekton.PipelineResource {
	return &tekton.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineResource",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Spec: tekton.PipelineResourceSpec{
			Type:   resourceType,
			Params: params,
		},
	}
}
