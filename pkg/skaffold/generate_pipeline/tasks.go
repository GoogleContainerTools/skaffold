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
	"errors"
	"fmt"
	"os"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

func generateBuildTasks(namespace string, configFiles []*ConfigFile) ([]*tekton.Task, error) {
	var tasks []*tekton.Task
	for i, configFile := range configFiles {
		task, err := generateBuildTask(configFile)
		if err != nil {
			return nil, err
		}
		task.Name = fmt.Sprintf("%s-%d", task.Name, i)

		if namespace != "" {
			nsFlag := []string{"--namespace", namespace}
			task.Spec.Steps[0].Args = append(task.Spec.Steps[0].Args, nsFlag...)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func generateBuildTask(configFile *ConfigFile) (*tekton.Task, error) {
	buildConfig := configFile.Profile.Build
	if len(buildConfig.Artifacts) == 0 {
		return nil, errors.New("no artifacts to build")
	}

	skaffoldVersion := os.Getenv("PIPELINE_SKAFFOLD_VERSION")
	if skaffoldVersion == "" {
		skaffoldVersion = version.Get().Version
	}

	resources := []tekton.TaskResource{
		{
			Name: "source",
			Type: tekton.PipelineResourceTypeGit,
		},
	}
	inputs := &tekton.Inputs{Resources: resources}
	outputs := &tekton.Outputs{Resources: resources}
	steps := []v1.Container{
		{
			Name:       "run-build",
			Image:      fmt.Sprintf("gcr.io/k8s-skaffold/skaffold:%s", skaffoldVersion),
			WorkingDir: "/workspace/source",
			Command:    []string{"skaffold", "build"},
			Args: []string{
				"--filename", configFile.Path,
				"--profile", "oncluster",
				"--file-output", "build.out",
			},
		},
	}

	// Add secret volume mounting if any artifacts in config need to be built with kaniko
	var volumes []v1.Volume
	for _, artifact := range buildConfig.Artifacts {
		if artifact.KanikoArtifact != nil {
			volumes = []v1.Volume{
				{
					Name: kanikoSecretName,
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: kanikoSecretName,
						},
					},
				},
			}
			steps[0].VolumeMounts = []v1.VolumeMount{
				{
					Name:      kanikoSecretName,
					MountPath: "/secret",
				},
			}
			steps[0].Env = []v1.EnvVar{
				{
					Name:  "GOOGLE_APPLICATION_CREDENTIALS",
					Value: "/secret/" + kanikoSecretName,
				},
			}
		}
	}

	return pipeline.NewTask("skaffold-build", inputs, outputs, steps, volumes), nil
}

func generateDeployTasks(namespace string, configFiles []*ConfigFile) ([]*tekton.Task, error) {
	var tasks []*tekton.Task
	for i, configFile := range configFiles {
		task, err := generateDeployTask(configFile)
		if err != nil {
			return nil, err
		}
		task.Name = fmt.Sprintf("%s-%d", task.Name, i)

		if namespace != "" {
			nsFlag := []string{"--namespace", namespace}
			task.Spec.Steps[0].Args = append(task.Spec.Steps[0].Args, nsFlag...)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func generateDeployTask(configFile *ConfigFile) (*tekton.Task, error) {
	deployConfig := configFile.Config.Deploy
	if deployConfig.HelmDeploy == nil && deployConfig.KubectlDeploy == nil && deployConfig.KustomizeDeploy == nil {
		return nil, errors.New("no Helm/Kubectl/Kustomize deploy config")
	}

	skaffoldVersion := os.Getenv("PIPELINE_SKAFFOLD_VERSION")
	if skaffoldVersion == "" {
		skaffoldVersion = version.Get().Version
	}

	resources := []tekton.TaskResource{
		{
			Name: "source",
			Type: tekton.PipelineResourceTypeGit,
		},
	}
	inputs := &tekton.Inputs{Resources: resources}
	steps := []v1.Container{
		{
			Name:       "run-deploy",
			Image:      fmt.Sprintf("gcr.io/k8s-skaffold/skaffold:%s", skaffoldVersion),
			WorkingDir: "/workspace/source",
			Command:    []string{"skaffold", "deploy"},
			Args: []string{
				"--filename", configFile.Path,
				"--build-artifacts", "build.out",
			},
		},
	}

	return pipeline.NewTask("skaffold-deploy", inputs, nil, steps, nil), nil
}
