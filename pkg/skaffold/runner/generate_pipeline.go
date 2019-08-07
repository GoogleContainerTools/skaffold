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

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	yamlv2 "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

var (
	// for testing
	reader = bufio.NewReader(os.Stdin)
)

func (r *SkaffoldRunner) GeneratePipeline(ctx context.Context, out io.Writer, config *latest.SkaffoldConfig, fileOut string) error {
	err := createSkaffoldProfile(out, config, r.runCtx.Opts.ConfigurationFile)
	if err != nil {
		return errors.Wrap(err, "setting up profile")
	}

	color.Default.Fprintln(out, "Generating Pipeline...")

	// Generate git resource for pipeline
	gitResource, err := generateGitResource()
	if err != nil {
		return errors.Wrap(err, "generating git resource for pipeline")
	}

	// Generate build task for pipeline
	var tasks []*tekton.Task
	taskBuild, err := generateBuildTask(config.Pipeline.Build)
	if err != nil {
		return errors.Wrap(err, "generating build task")
	}
	tasks = append(tasks, taskBuild)

	// Generate deploy task for pipeline
	taskDeploy, err := generateDeployTask(config.Pipeline.Deploy)
	if err != nil {
		return errors.Wrap(err, "generating deploy task")
	}
	tasks = append(tasks, taskDeploy)

	// Generate pipeline from git resource and tasks
	pipeline, err := generatePipeline(tasks)
	if err != nil {
		return errors.Wrap(err, "generating tekton pipeline")
	}

	// json.Marshal all pieces of pipeline, then convert all jsons to yamls
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

	var output bytes.Buffer
	for _, item := range jsons {
		itemYaml, err := yaml.JSONToYAML(item)
		if err != nil {
			return errors.Wrap(err, "converting jsons to yamls")
		}
		output.Write(append(itemYaml, []byte("---\n")...))
	}

	// write all yaml pieces to output
	return ioutil.WriteFile(fileOut, output.Bytes(), 0755)
}

func generateGitResource() (*tekton.PipelineResource, error) {
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

func generateBuildTask(buildConfig latest.BuildConfig) (*tekton.Task, error) {
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

func generateDeployTask(deployConfig latest.DeployConfig) (*tekton.Task, error) {
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

func createSkaffoldProfile(out io.Writer, config *latest.SkaffoldConfig, configFile string) error {
	color.Default.Fprintln(out, "Checking for oncluster skaffold profile...")
	profileExists := false
	for _, profile := range config.Profiles {
		if profile.Name == "oncluster" {
			profileExists = true
			break
		}
	}

	// Check for existing oncluster profile, if none exists then prompt to create one
	if profileExists {
		color.Default.Fprintln(out, "profile \"oncluster\" found!")
		return nil
	}

confirmLoop:
	for {
		color.Default.Fprintf(out, "No profile \"oncluster\" found. Create one? [y/n]: ")
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

	color.Default.Fprintln(out, "Creating skaffold profile \"oncluster\"...")
	profile, err := generateProfile(out, config)
	if err != nil {
		return errors.Wrap(err, "generating profile \"oncluster\"")
	}

	bProfile, err := yamlv2.Marshal([]*latest.Profile{profile})
	if err != nil {
		return errors.Wrap(err, "marshaling new profile")
	}

	fileContents, err := ioutil.ReadFile(configFile)
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
	fileStrings[profilePos] = strings.TrimSpace(string(bProfile))

	fileContents = []byte((strings.Join(fileStrings, "\n")))

	if err := ioutil.WriteFile(configFile, fileContents, 0644); err != nil {
		return errors.Wrap(err, "writing profile to skaffold config")
	}

	return nil
}

func generateProfile(out io.Writer, config *latest.SkaffoldConfig) (*latest.Profile, error) {
	if len(config.Build.Artifacts) == 0 {
		return nil, errors.New("No Artifacts to add to profile")
	}

	profile := &latest.Profile{
		Name: "oncluster",
		Pipeline: latest.Pipeline{
			Build:  config.Pipeline.Build,
			Deploy: latest.DeployConfig{},
		},
	}
	profile.Build.Cluster = &latest.ClusterDetails{
		PullSecretName: "kaniko-secret",
	}
	profile.Build.LocalBuild = nil
	// Add kaniko build config for artifacts
	for _, artifact := range profile.Build.Artifacts {
		artifact.ImageName = fmt.Sprintf("%s-pipeline", artifact.ImageName)
		if artifact.DockerArtifact != nil {
			color.Default.Fprintf(out, "Cannot use Docker to build %s on cluster. Adding config for building with Kaniko.\n", artifact.ImageName)
			artifact.DockerArtifact = nil
			artifact.KanikoArtifact = &latest.KanikoArtifact{
				BuildContext: &latest.KanikoBuildContext{
					GCSBucket: "skaffold-kaniko",
				},
			}
		}
	}

	return profile, nil
}
