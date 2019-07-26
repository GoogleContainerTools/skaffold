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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	yamlv2 "gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *SkaffoldRunner) GeneratePipeline(ctx context.Context, out io.Writer, config *latest.SkaffoldConfig, fileOut string) error {
	if config.APIVersion == "" || config.Kind == "" {
		return errors.New("Invalid skaffold config")
	}

	reader := bufio.NewReader(os.Stdin)
	err := createSkaffoldProfile(config, reader)
	if err != nil {
		return errors.Wrap(err, "setting up profile")
	}

	fmt.Println("Generating Pipeline...")

	var output bytes.Buffer
	gitResource, err := generateGitResource(config)
	if err != nil {
		fmt.Print("Could not get git url automatically, please enter: ")
		newUrl, err := reader.ReadString('\n')
		if err != nil {
			return errors.Wrap(err, "generating git resource for pipeline")
		}
		gitResource.Spec.Params[0].Value = newUrl
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
	//TODO: Ensure that tasks run in order
	// Create tasks in pipeline spec for all corresponding tasks
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
			pipelineTask.RunAfter = []string{pipeline.Spec.Tasks[i-1].Name}
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
	}, errors.Wrap(err, "getting git repo from git config")
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
					Name:       "run-build",
					Image:      "gcr.io/k8s-skaffold/skaffold:v0.34.0",
					WorkingDir: "/workspace/source",
					Command:    []string{"skaffold"},
					Args: []string{"build",
						"--filename", "skaffold.yaml",
						"--profile", "oncluster",
						"--file-output", "build.out",
					},
				},
			},
		},
	}, nil
}

func generateDeployTask(config *latest.SkaffoldConfig) (tekton.Task, error) {
	deployConfig := config.Pipeline.Deploy
	if deployConfig.HelmDeploy == nil && deployConfig.KubectlDeploy == nil && deployConfig.KustomizeDeploy == nil {
		return tekton.Task{}, errors.New("No Helm/Kubectl/Kustomize deploy config")
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
					Name:       "run-deploy",
					Image:      "gcr.io/k8s-skaffold/skaffold:v0.34.0",
					WorkingDir: "/workspace/source",
					Command:    []string{"skaffold"},
					Args: []string{
						"deploy",
						"--filename", "skaffold.yaml",
						"--profile", "oncluster",
						"--build-artifacts", "build.out",
					},
				},
			},
		},
	}, nil
}

func createSkaffoldProfile(config *latest.SkaffoldConfig, reader *bufio.Reader) error {
	fmt.Println("Checking for oncluster skaffold profile...")
	profileExists := false
	for _, profile := range config.Profiles {
		if profile.Name == "oncluster" {
			profileExists = true
			break
		}
	}

	// Check for existing oncluster profile, if none exists then prompt to create one
	if profileExists {
		fmt.Println("profile \"oncluster\" found!")
		return nil
	} else {
	confirmLoop:
		for {
			fmt.Print("No profile \"oncluster\" found. Create one? [y/n]: ")
			response, err := reader.ReadString('\n')
			if err != nil {
				return errors.Wrap(err, "reading user confirmation")
			}

			response = strings.ToLower(strings.TrimSpace(response))
			switch response {
			case "y", "yes":
				break confirmLoop
			case "n", "no":
				return nil
			}
		}
	}

	fmt.Println("Creating skaffold profile \"oncluster\"...")
	newProfile := []latest.Profile{
		{
			Name: "oncluster",
			Pipeline: latest.Pipeline{
				Build:  config.Pipeline.Build,
				Deploy: latest.DeployConfig{},
			},
		},
	}
	newProfile[0].Build.Cluster = &latest.ClusterDetails{
		PullSecretName: "kaniko-secret",
	}
	newProfile[0].Build.LocalBuild = nil
	// Add kaniko build config for artifacts
	for _, artifact := range newProfile[0].Build.Artifacts {
		artifact.ImageName = fmt.Sprintf("%s-pipeline", artifact.ImageName)
		if artifact.DockerArtifact != nil {
			fmt.Printf("Cannot use Docker to build %s on cluster. Adding config for building with Kaniko.\n", artifact.ImageName)
			artifact.DockerArtifact = nil
			artifact.KanikoArtifact = &latest.KanikoArtifact{
				BuildContext: &latest.KanikoBuildContext{
					GCSBucket: "skaffold-kaniko",
				},
			}
		}
	}

	bNewProfile, err := yamlv2.Marshal(newProfile)
	if err != nil {
		return errors.Wrap(err, "marshaling new profile")
	}

	fileContents, err := ioutil.ReadFile("skaffold.yaml")
	if err != nil {
		return errors.Wrap(err, "reading file contents")
	}
	fileStrings := strings.Split(strings.TrimSpace(string(fileContents)), "\n")

	var profilePos int
	if len(config.Profiles) == 0 {
		// Create new profiles section
		fileStrings = append(fileStrings, "profiles:")
		profilePos = len(fileStrings)
	} else {
		for i, line := range fileStrings {
			if line == "profiles:" {
				profilePos = i + 1
			}
		}
	}

	fileStrings = append(fileStrings, "")
	copy(fileStrings[profilePos+1:], fileStrings[profilePos:])
	fileStrings[profilePos] = strings.TrimSpace(string(bNewProfile))

	fileContents = []byte((strings.Join(fileStrings, "\n")))

	if err := ioutil.WriteFile("skaffold.yaml", fileContents, 0644); err != nil {
		return errors.Wrap(err, "writing profile to skaffold.yaml")
	}

	return nil
}
