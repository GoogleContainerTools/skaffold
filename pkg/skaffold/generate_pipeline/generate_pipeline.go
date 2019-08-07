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
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func GenerateGitResource() (*tekton.PipelineResource, error) {
	// Get git repo url
	gitURL := os.Getenv("PIPELINE_GIT_URL")
	if gitURL == "" {
		getGitRepo := exec.Command("git", "config", "--get", "remote.origin.url")
		bGitRepo, err := getGitRepo.Output()
		if err != nil {
			return nil, errors.Wrap(err, "getting git repo from git config")
		}
		gitURL = string(bGitRepo)
	}

	return pipeline.NewGitResource("source-git", gitURL), nil
}

func GenerateBuildTask(buildConfig latest.BuildConfig) (*tekton.Task, error) {
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
	steps := []corev1.Container{
		{
			Name:       "run-build",
			Image:      fmt.Sprintf("gcr.io/k8s-skaffold/skaffold:%s", skaffoldVersion),
			WorkingDir: "/workspace/source",
			Command:    []string{"skaffold"},
			Args: []string{"build",
				"--filename", "skaffold.yaml",
				"--profile", "oncluster",
				"--file-output", "build.out",
			},
		},
	}

	return pipeline.NewTask("skaffold-build", resources, steps), nil
}

func GenerateDeployTask(deployConfig latest.DeployConfig) (*tekton.Task, error) {
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
	steps := []corev1.Container{
		{
			Name:       "run-deploy",
			Image:      fmt.Sprintf("gcr.io/k8s-skaffold/skaffold:%s", skaffoldVersion),
			WorkingDir: "/workspace/source",
			Command:    []string{"skaffold"},
			Args: []string{
				"deploy",
				"--filename", "skaffold.yaml",
				"--profile", "oncluster",
				"--build-artifacts", "build.out",
			},
		},
	}

	return pipeline.NewTask("skaffold-deploy", resources, steps), nil
}

func GeneratePipeline(tasks []*tekton.Task) (*tekton.Pipeline, error) {
	if len(tasks) == 0 {
		return nil, errors.New("no tasks to add to pipeline")
	}

	resources := []tekton.PipelineDeclaredResource{
		{
			Name: "source-repo",
			Type: tekton.PipelineResourceTypeGit,
		},
	}
	// Create tasks in pipeline spec for all corresponding tasks
	pipelineTasks := make([]tekton.PipelineTask, 0)
	for i, task := range tasks {
		pipelineTask := tekton.PipelineTask{
			Name: fmt.Sprintf("%s-task", task.Name),
			TaskRef: tekton.TaskRef{
				Name: task.Name,
			},
			RunAfter: []string{},
			Resources: &tekton.PipelineTaskResources{
				Inputs: []tekton.PipelineTaskInputResource{
					{
						Name:     "source",
						Resource: "source-repo",
					},
				},
			},
		}
		if i > 0 {
			pipelineTask.RunAfter = []string{pipelineTasks[i-1].Name}
		}
		pipelineTasks = append(pipelineTasks, pipelineTask)
	}

	return pipeline.NewPipeline("skaffold-pipeline", resources, pipelineTasks), nil
}
