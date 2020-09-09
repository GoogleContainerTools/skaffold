/*
Copyright 2020 The Skaffold Authors

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

package docker

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	EvalBuildArgs = evalBuildArgs // To override during testing

	// default build args for skaffold non-debug mode
	nonDebugModeArgs = map[string]string{}
	// default build args for skaffold debug mode
	debugModeArgs = map[string]string{
		"SKAFFOLD_GO_GCFLAGS": "'all=-N -l'", // disable build optimization for Golang
		// TODO: Add for other languages
	}
)

func evalBuildArgs(mode config.RunMode, workspace string, a *latest.DockerArtifact) (map[string]*string, error) {
	var defaults map[string]string
	switch mode {
	case config.RunModes.Debug:
		defaults = debugModeArgs
	default:
		defaults = nonDebugModeArgs
	}
	result := map[string]*string{
		"SKAFFOLD_RUN_MODE": util.StringPtr(string(mode)),
	}
	for k, v := range defaults {
		result[k] = &v
	}
	absDockerfilePath, err := NormalizeDockerfilePath(workspace, a.DockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("normalizing dockerfile path: %w", err)
	}
	f, err := os.Open(absDockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("reading dockerfile: %w", err)
	}
	defer f.Close()
	result, err = filterUnusedBuildArgs(f, result)
	if err != nil {
		return nil, fmt.Errorf("removing unused default args: %w", err)
	}
	for k, v := range a.BuildArgs {
		result[k] = v
	}
	result, err = util.EvaluateEnvTemplateMap(result)
	if err != nil {
		return nil, fmt.Errorf("unable to expand build args: %w", err)
	}
	return result, nil
}
