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
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *SkaffoldRunner) GeneratePipeline(ctx context.Context, out io.Writer, config *latest.SkaffoldConfig, fileOut string) error {
	if config.APIVersion == "" || config.Kind == "" {
		return errors.New("Invalid skaffold config")
	}

	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	defer encoder.Close()

	// Get git repo url
	getGitRepo := exec.Command("git", "config", "--get", "remote.origin.url")
	gitRepo, err := getGitRepo.Output()
	if err != nil {
		return errors.Wrap(err, "getting git repo from git config")
	}

	// Create git resource for pipeline from users current git repo
	gitResource := tekton.PipelineResource{
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
	}

	//TODO(marlon-gamez): Check if config.pipeline.Build.Artifacts exists

	// Create build tasks for all artifacts listed in skaffold build config
	var tasks []tekton.Task
	for i := range config.Pipeline.Build.Artifacts {
		taskBuild := tekton.Task{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Task",
				APIVersion: "tekton.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("skaffold-build-%d", i),
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
		}
		tasks = append(tasks, taskBuild)
	}

	//TODO(marlon-gamez): Check if config.pipeline.Deploy.HelmDeploy/KubectlDeploy/KustomizeDeploy exist
	taskDeploy := tekton.Task{
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
	}
	tasks = append(tasks, taskDeploy)

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
			Tasks: []tekton.PipelineTask{
				{
					Name: "skaffold-build-task",
					TaskRef: tekton.TaskRef{
						Name: "skaffold-build",
					},
					Resources: &tekton.PipelineTaskResources{
						Inputs: []tekton.PipelineTaskInputResource{
							{
								Name:     "source",
								Resource: "source-repo",
							},
						},
					},
				},
			},
		},
	}
	// Create tasks in pipeline spec for all corresponding tasks
	for i, task := range tasks {
		pipelineTask := tekton.PipelineTask{
			Name: fmt.Sprintf("skaffold-build-task-%d", i),
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

	// Encode all pieces of pipeline to out buffer then write buffer to file
	if err := encoder.Encode(gitResource); err != nil {
		return errors.Wrap(err, "encoding git resource")
	}
	for _, task := range tasks {
		if err := encoder.Encode(task); err != nil {
			return errors.Wrap(err, "encoding task")
		}
	}
	if err := encoder.Encode(pipeline); err != nil {
		return errors.Wrap(err, "encoding pipeline")
	}
	if err := ioutil.WriteFile(fileOut, output.Bytes(), 0755); err != nil {
		return err
	}
	return nil

}
