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
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	pipeline "github.com/GoogleContainerTools/skaffold/pkg/skaffold/generate_pipeline"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

func (r *SkaffoldRunner) GeneratePipeline(ctx context.Context, out io.Writer, config *latest.SkaffoldConfig, fileOut string) error {
	err := pipeline.CreateSkaffoldProfile(out, config, r.runCtx.Opts.ConfigurationFile)
	if err != nil {
		return errors.Wrap(err, "setting up profile")
	}

	color.Default.Fprintln(out, "Generating Pipeline...")

	// Generate git resource for pipeline
	gitResource, err := pipeline.GenerateGitResource()
	if err != nil {
		return errors.Wrap(err, "generating git resource for pipeline")
	}

	// Generate build task for pipeline
	var tasks []*tekton.Task
	taskBuild, err := pipeline.GenerateBuildTask(config.Pipeline.Build)
	if err != nil {
		return errors.Wrap(err, "generating build task")
	}
	tasks = append(tasks, taskBuild)

	// Generate deploy task for pipeline
	taskDeploy, err := pipeline.GenerateDeployTask(config.Pipeline.Deploy)
	if err != nil {
		return errors.Wrap(err, "generating deploy task")
	}
	tasks = append(tasks, taskDeploy)

	// Generate pipeline from git resource and tasks
	pipeline, err := pipeline.GeneratePipeline(tasks)
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
