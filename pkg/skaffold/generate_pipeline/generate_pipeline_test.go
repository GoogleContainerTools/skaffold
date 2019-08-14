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

package generatepipeline

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGeneratePipeline(t *testing.T) {
	var tests = []struct {
		description      string
		tasks            []*tekton.Task
		expectedPipeline *tekton.Pipeline
		shouldErr        bool
	}{
		{
			description: "successful tekton pipeline generation",
			tasks: []*tekton.Task{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-build",
					},
				},
			},
			expectedPipeline: &tekton.Pipeline{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pipeline",
					APIVersion: "tekton.dev/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "skaffold-pipeline",
				},
				Spec: tekton.PipelineSpec{
					Resources: []tekton.PipelineDeclaredResource{
						{
							Name: "source-repo",
							Type: tekton.PipelineResourceTypeGit,
						},
					},
					Tasks: []tekton.PipelineTask{
						{
							Name: "test-build-task",
							TaskRef: tekton.TaskRef{
								Name: "test-build",
							},
							Resources: &tekton.PipelineTaskResources{
								Inputs: []tekton.PipelineTaskInputResource{
									{
										Name:     "source",
										Resource: "source-repo",
									},
								},
								Outputs: []tekton.PipelineTaskOutputResource{
									{
										Name:     "source",
										Resource: "source-repo",
									},
								},
							},
						},
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "successful multiple tasks",
			tasks: []*tekton.Task{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-build",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-deploy",
					},
				},
			},
			expectedPipeline: &tekton.Pipeline{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pipeline",
					APIVersion: "tekton.dev/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "skaffold-pipeline",
				},
				Spec: tekton.PipelineSpec{
					Resources: []tekton.PipelineDeclaredResource{
						{
							Name: "source-repo",
							Type: tekton.PipelineResourceTypeGit,
						},
					},
					Tasks: []tekton.PipelineTask{
						{
							Name: "test-build-task",
							TaskRef: tekton.TaskRef{
								Name: "test-build",
							},
							Resources: &tekton.PipelineTaskResources{
								Inputs: []tekton.PipelineTaskInputResource{
									{
										Name:     "source",
										Resource: "source-repo",
									},
								},
								Outputs: []tekton.PipelineTaskOutputResource{
									{
										Name:     "source",
										Resource: "source-repo",
									},
								},
							},
						},
						{
							Name: "test-deploy-task",
							TaskRef: tekton.TaskRef{
								Name: "test-deploy",
							},
							Resources: &tekton.PipelineTaskResources{
								Inputs: []tekton.PipelineTaskInputResource{
									{
										Name:     "source",
										Resource: "source-repo",
										From:     []string{"test-build-task"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "fail generating tekton pipeline",
			tasks:       []*tekton.Task{},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pipeline, err := generatePipeline(test.tasks)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedPipeline, pipeline)
		})
	}
}

func TestGenerateBuildTask(t *testing.T) {
	var tests = []struct {
		description string
		buildConfig latest.BuildConfig
		shouldErr   bool
	}{
		{
			description: "successfully generate build task",
			buildConfig: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{
						ImageName: "testArtifact",
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating build task",
			buildConfig: latest.BuildConfig{
				Artifacts: []*latest.Artifact{},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := generateBuildTask(test.buildConfig)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateDeployTask(t *testing.T) {
	var tests = []struct {
		description  string
		deployConfig latest.DeployConfig
		shouldErr    bool
	}{
		{
			description: "successfully generate deploy task",
			deployConfig: latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy: &latest.HelmDeploy{},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating deploy task",
			deployConfig: latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy:      nil,
					KubectlDeploy:   nil,
					KustomizeDeploy: nil,
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := generateDeployTask(test.deployConfig)
			t.CheckError(test.shouldErr, err)
		})
	}
}
