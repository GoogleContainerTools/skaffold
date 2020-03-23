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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ghodss/yaml"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// ConfigFile keeps track of config files and their corresponding SkaffoldConfigs and generated Profiles
type ConfigFile struct {
	Path    string
	Config  *latest.SkaffoldConfig
	Profile *latest.Profile
}

func Yaml(out io.Writer, runCtx *runcontext.RunContext, configFiles []*ConfigFile) (*bytes.Buffer, error) {
	// Generate git resource for pipeline
	gitResource, err := generateGitResource()
	if err != nil {
		return nil, fmt.Errorf("generating git resource for pipeline: %w", err)
	}

	// Generate build task for pipeline
	var tasks []*tekton.Task
	buildTasks, err := generateBuildTasks(runCtx.Opts.Namespace, configFiles)
	if err != nil {
		return nil, fmt.Errorf("generating build task: %w", err)
	}
	tasks = append(tasks, buildTasks...)

	// Generate deploy task for pipeline
	deployTasks, err := generateDeployTasks(runCtx.Opts.Namespace, configFiles)
	if err != nil {
		return nil, fmt.Errorf("generating deploy task: %w", err)
	}
	tasks = append(tasks, deployTasks...)

	// Generate pipeline from git resource and tasks
	pipeline, err := generatePipeline(tasks)
	if err != nil {
		return nil, fmt.Errorf("generating tekton pipeline: %w", err)
	}

	// json.Marshal all pieces of pipeline, then convert all jsons to yamls
	var jsons [][]byte
	bGitResource, err := json.Marshal(gitResource)
	if err != nil {
		return nil, fmt.Errorf("marshaling git resource: %w", err)
	}
	jsons = append(jsons, bGitResource)
	for _, task := range tasks {
		bTask, err := json.Marshal(task)
		if err != nil {
			return nil, fmt.Errorf("marshaling task: %w", err)
		}
		jsons = append(jsons, bTask)
	}
	bPipeline, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshaling pipeline: %w", err)
	}
	jsons = append(jsons, bPipeline)

	output := bytes.NewBuffer([]byte{})
	for _, item := range jsons {
		itemYaml, err := yaml.JSONToYAML(item)
		if err != nil {
			return nil, fmt.Errorf("converting jsons to yamls: %w", err)
		}
		output.Write(append(itemYaml, []byte("---\n")...))
	}
	return output, nil
}

func generateGitResource() (*tekton.PipelineResource, error) {
	// Get git repo url
	gitURL := os.Getenv("PIPELINE_GIT_URL")
	if gitURL == "" {
		getGitRepo := exec.Command("git", "config", "--get", "remote.origin.url")
		bGitRepo, err := getGitRepo.Output()
		if err != nil {
			return nil, fmt.Errorf("getting git repo from git config: %w", err)
		}
		gitURL = string(bGitRepo)
	}

	return pipeline.NewGitResource("source-git", gitURL), nil
}

func generatePipeline(tasks []*tekton.Task) (*tekton.Pipeline, error) {
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
		// Add output resource for build tasks, input for deploy task
		if strings.Contains(task.Name, "build") {
			pipelineTask.Resources.Outputs = []tekton.PipelineTaskOutputResource{
				{
					Name:     "source",
					Resource: "source-repo",
				},
			}
		} else {
			// Get the git resource for deploy commands from their corresponding build command
			from := strings.Replace(pipelineTask.Name, "deploy", "build", 1)
			pipelineTask.Resources.Inputs[0].From = []string{from}
		}

		pipelineTasks = append(pipelineTasks, pipelineTask)
	}

	return pipeline.NewPipeline("skaffold-pipeline", resources, pipelineTasks), nil
}
