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

package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	//"gopkg.in/yaml.v2"
	"github.com/ghodss/yaml"
	"k8s.io/api/core/v1"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *SkaffoldRunner) GeneratePipeline(ctx context.Context, out io.Writer, config *latest.SkaffoldConfig, fileOut string) error {
	if config.APIVersion == "" || config.Kind == "" {
		return errors.New("Invalid skaffold config")
	}

	var output bytes.Buffer

	gitResource, err := generateGitResource(config)
	if err != nil {
		return errors.Wrap(err, "generating git resource for pipeline")
	}

	var tasks []tekton.Task
	taskBuild, err := generateBuildTask(config)
	if err == nil {
		tasks = append(tasks, taskBuild)
	}
	taskDeploy, err := generateDeployTask(config)
	if err == nil {
		tasks = append(tasks, taskDeploy)
	}

	// Create pipeline to tie together all tasks
	pipeline := tekton.Pipeline{
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
			Tasks: []tekton.PipelineTask{},
		},
	}
	// Create tasks in pipeline spec for all corresponding tasks
	for _, task := range tasks {
		pipelineTask := tekton.PipelineTask{
			Name: fmt.Sprintf("%s-task", task.Name),
			TaskRef: tekton.TaskRef{
				Name: task.Name,
			},
			Resources: &tekton.PipelineTaskResources{
				Inputs: []tekton.PipelineTaskInputResource{
					{
						Name:     "source",
						Resource: "source-repo",
					},
				},
			},
		}
		pipeline.Spec.Tasks = append(pipeline.Spec.Tasks, pipelineTask)
	}

	// json.Marshal all pieces of pipeline, then convert all to yaml and write them to file
	var jsons [][]byte

	bGitResource, err := json.Marshal(gitResource)
	if err != nil {
		return errors.Wrap(err, "marshaling git resource")
	}
	jsons = append(jsons, bGitResource)
	for _, task := range tasks {
		bTask, err := json.Marshal(task)
		if err != nil {
			return errors.Wrap(err, "marshaling task")
		}
		jsons = append(jsons, bTask)
	}
	bPipeline, err := json.Marshal(pipeline)
	if err != nil {
		return errors.Wrap(err, "marshaling pipeline")
	}
	jsons = append(jsons, bPipeline)

	for _, item := range jsons {
		itemYaml, err := yaml.JSONToYAML(item)
		if err != nil {
			return errors.Wrap(err, "converting json to yaml")
		}
		output.Write(append(itemYaml, []byte("---\n")...))
	}
	if err := ioutil.WriteFile(fileOut, output.Bytes(), 0755); err != nil {
		return err
	}
	return nil

}

func generateGitResource(config *latest.SkaffoldConfig) (tekton.PipelineResource, error) {
	// Get git repo url
	getGitRepo := exec.Command("git", "config", "--get", "remote.origin.url")
	gitRepo, err := getGitRepo.Output()
	if err != nil {
		return tekton.PipelineResource{}, errors.Wrap(err, "getting git repo from git config")
	}

	// Create git resource for pipeline from users current git repo
	return tekton.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineResource",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "source-git",
		},
		Spec: tekton.PipelineResourceSpec{
			Type: tekton.PipelineResourceTypeGit,
			Params: []tekton.Param{
				{
					Name:  "url",
					Value: string(gitRepo),
				},
			},
		},
	}, nil
}

func generateBuildTask(config *latest.SkaffoldConfig) (tekton.Task, error) {
	if len(config.Pipeline.Build.Artifacts) == 0 {
		return tekton.Task{}, errors.New("No artifacts to build")
	}
	return tekton.Task{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Task",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "skaffold-build",
		},
		Spec: tekton.TaskSpec{
			Inputs: &tekton.Inputs{
				Resources: []tekton.TaskResource{
					{
						Name: "source",
						Type: tekton.PipelineResourceTypeGit,
					},
				},
			},
			Steps: []v1.Container{
				{
					Name:    "run-build",
					Image:   "gcr.io/k8s-skaffold/skaffold:v0.33.0",
					Command: []string{"skaffold build"},
					Args:    []string{"--filename", "/workspace/source/pipeline/skaffold.yaml"},
				},
			},
		},
	}, nil
}

func generateDeployTask(config *latest.SkaffoldConfig) (tekton.Task, error) {
	deployConfig := config.Pipeline.Deploy
	if deployConfig.HelmDeploy == nil && deployConfig.KubectlDeploy == nil && deployConfig.KustomizeDeploy == nil {
		return tekton.Task{}, errors.New("No Help/Kubectl/Kustomize deploy config")
	}

	return tekton.Task{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Task",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "skaffold-deploy",
		},
		Spec: tekton.TaskSpec{
			Inputs: &tekton.Inputs{
				Resources: []tekton.TaskResource{
					{
						Name: "source",
						Type: tekton.PipelineResourceTypeGit,
					},
				},
			},
			Steps: []v1.Container{
				{
					Name:    "run-deploy",
					Image:   "gcr.io/k8s-skaffold/skaffold:v0.33.0",
					Command: []string{"skaffold deploy"},
					Args:    []string{"--filename", "/workspace/source/pipeline/skaffold.yaml"},
				},
			},
		},
	}, nil
}
