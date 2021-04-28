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

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateBuildTasks(t *testing.T) {
	var tests = []struct {
		description   string
		configFiles   []*ConfigFile
		shouldErr     bool
		namespace     string
		expectedTasks []*tekton.Task
	}{
		{
			description: "successfully generate build tasks",
			configFiles: []*ConfigFile{
				{
					Path: "test1",
					Profile: &latest_v1.Profile{
						Pipeline: latest_v1.Pipeline{
							Build: latest_v1.BuildConfig{
								Artifacts: []*latest_v1.Artifact{
									{
										ImageName: "testArtifact1",
									},
								},
							},
						},
					},
				},
				{
					Path: "test2",
					Profile: &latest_v1.Profile{
						Pipeline: latest_v1.Pipeline{
							Build: latest_v1.BuildConfig{
								Artifacts: []*latest_v1.Artifact{
									{
										ImageName: "testArtifact2",
									},
								},
							},
						},
					},
				},
			},
			namespace: "",
			shouldErr: false,
			expectedTasks: []*tekton.Task{
				{
					TypeMeta:   metav1.TypeMeta{Kind: "Task", APIVersion: "tekton.dev/v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "skaffold-build-0"},
					Spec: tekton.TaskSpec{
						Inputs:  &tekton.Inputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Outputs: &tekton.Outputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Steps: []v1.Container{
							{
								Name:       "run-build",
								Image:      "gcr.io/k8s-skaffold/skaffold:",
								Command:    []string{"skaffold", "build"},
								Args:       []string{"--filename", "test1", "--profile", "oncluster", "--file-output", "build.out"},
								WorkingDir: "/workspace/source",
							},
						},
					},
				},
				{
					TypeMeta:   metav1.TypeMeta{Kind: "Task", APIVersion: "tekton.dev/v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "skaffold-build-1"},
					Spec: tekton.TaskSpec{
						Inputs:  &tekton.Inputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Outputs: &tekton.Outputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Steps: []v1.Container{
							{
								Name:       "run-build",
								Image:      "gcr.io/k8s-skaffold/skaffold:",
								Command:    []string{"skaffold", "build"},
								Args:       []string{"--filename", "test2", "--profile", "oncluster", "--file-output", "build.out"},
								WorkingDir: "/workspace/source",
							},
						},
					},
				},
			},
		},
		{
			description: "build task with namespace",
			configFiles: []*ConfigFile{
				{
					Path: "test1",
					Profile: &latest_v1.Profile{
						Pipeline: latest_v1.Pipeline{
							Build: latest_v1.BuildConfig{
								Artifacts: []*latest_v1.Artifact{
									{
										ImageName: "testArtifact1",
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			shouldErr: false,
			expectedTasks: []*tekton.Task{
				{
					TypeMeta:   metav1.TypeMeta{Kind: "Task", APIVersion: "tekton.dev/v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "skaffold-build-0"},
					Spec: tekton.TaskSpec{
						Inputs:  &tekton.Inputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Outputs: &tekton.Outputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Steps: []v1.Container{
							{
								Name:    "run-build",
								Image:   "gcr.io/k8s-skaffold/skaffold:",
								Command: []string{"skaffold", "build"},
								Args: []string{
									"--filename",
									"test1",
									"--profile",
									"oncluster",
									"--file-output",
									"build.out",
									"--namespace",
									"test-ns",
								},
								WorkingDir: "/workspace/source",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got, err := generateBuildTasks(test.namespace, test.configFiles)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedTasks, got)
		})
	}
}

func TestGenerateBuildTask(t *testing.T) {
	var tests = []struct {
		description string
		buildConfig latest_v1.BuildConfig
		shouldErr   bool
	}{
		{
			description: "successfully generate build task",
			buildConfig: latest_v1.BuildConfig{
				Artifacts: []*latest_v1.Artifact{
					{
						ImageName: "testArtifact",
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating build task",
			buildConfig: latest_v1.BuildConfig{
				Artifacts: []*latest_v1.Artifact{},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configFile := &ConfigFile{
				Path: "test",
				Profile: &latest_v1.Profile{
					Pipeline: latest_v1.Pipeline{
						Build: test.buildConfig,
					},
				},
			}
			_, err := generateBuildTask(configFile)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateDeployTasks(t *testing.T) {
	var tests = []struct {
		description   string
		configFiles   []*ConfigFile
		shouldErr     bool
		namespace     string
		expectedTasks []*tekton.Task
	}{
		{
			description: "successfully generate deploy tasks",
			configFiles: []*ConfigFile{
				{
					Path: "test1",
					Config: &latest_v1.SkaffoldConfig{
						Pipeline: latest_v1.Pipeline{
							Deploy: latest_v1.DeployConfig{
								DeployType: latest_v1.DeployType{
									HelmDeploy: &latest_v1.HelmDeploy{},
								},
							},
						},
					},
				},
				{
					Path: "test2",
					Config: &latest_v1.SkaffoldConfig{
						Pipeline: latest_v1.Pipeline{
							Deploy: latest_v1.DeployConfig{
								DeployType: latest_v1.DeployType{
									HelmDeploy: &latest_v1.HelmDeploy{},
								},
							},
						},
					},
				},
			},
			namespace: "",
			shouldErr: false,
			expectedTasks: []*tekton.Task{
				{
					TypeMeta:   metav1.TypeMeta{Kind: "Task", APIVersion: "tekton.dev/v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "skaffold-deploy-0"},
					Spec: tekton.TaskSpec{
						Inputs: &tekton.Inputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Steps: []v1.Container{
							{
								Name:       "run-deploy",
								Image:      "gcr.io/k8s-skaffold/skaffold:",
								Command:    []string{"skaffold", "deploy"},
								Args:       []string{"--filename", "test1", "--build-artifacts", "build.out"},
								WorkingDir: "/workspace/source",
							},
						},
					},
				},
				{
					TypeMeta:   metav1.TypeMeta{Kind: "Task", APIVersion: "tekton.dev/v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "skaffold-deploy-1"},
					Spec: tekton.TaskSpec{
						Inputs: &tekton.Inputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Steps: []v1.Container{
							{
								Name:       "run-deploy",
								Image:      "gcr.io/k8s-skaffold/skaffold:",
								Command:    []string{"skaffold", "deploy"},
								Args:       []string{"--filename", "test2", "--build-artifacts", "build.out"},
								WorkingDir: "/workspace/source",
							},
						},
					},
				},
			},
		},
		{
			description: "deploy task with namespace",
			configFiles: []*ConfigFile{
				{
					Path: "test1",
					Config: &latest_v1.SkaffoldConfig{
						Pipeline: latest_v1.Pipeline{
							Deploy: latest_v1.DeployConfig{
								DeployType: latest_v1.DeployType{
									HelmDeploy: &latest_v1.HelmDeploy{},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			shouldErr: false,
			expectedTasks: []*tekton.Task{
				{
					TypeMeta:   metav1.TypeMeta{Kind: "Task", APIVersion: "tekton.dev/v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "skaffold-deploy-0"},
					Spec: tekton.TaskSpec{
						Inputs: &tekton.Inputs{Resources: []tekton.TaskResource{{Name: "source", Type: "git"}}},
						Steps: []v1.Container{
							{
								Name:    "run-deploy",
								Image:   "gcr.io/k8s-skaffold/skaffold:",
								Command: []string{"skaffold", "deploy"},
								Args: []string{
									"--filename",
									"test1",
									"--build-artifacts",
									"build.out",
									"--namespace",
									"test-ns",
								},
								WorkingDir: "/workspace/source",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got, err := generateDeployTasks(test.namespace, test.configFiles)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedTasks, got)
		})
	}
}

func TestGenerateDeployTask(t *testing.T) {
	var tests = []struct {
		description  string
		deployConfig latest_v1.DeployConfig
		shouldErr    bool
	}{
		{
			description: "successfully generate deploy task",
			deployConfig: latest_v1.DeployConfig{
				DeployType: latest_v1.DeployType{
					HelmDeploy: &latest_v1.HelmDeploy{},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating deploy task",
			deployConfig: latest_v1.DeployConfig{
				DeployType: latest_v1.DeployType{
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
			configFile := &ConfigFile{
				Path: "test",
				Config: &latest_v1.SkaffoldConfig{
					Pipeline: latest_v1.Pipeline{
						Deploy: test.deployConfig,
					},
				},
			}

			_, err := generateDeployTask(configFile)
			t.CheckError(test.shouldErr, err)
		})
	}
}
