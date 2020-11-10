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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	// EvalBuildArgsWithEnv evaluates the build args provided in the artifact definition and adds additional runtime defaults and extra arguments, based on OS and custom environment variables
	EvalBuildArgsWithEnv = evalBuildArgs

	// EvalBuildArgs evaluates the build args provided in the artifact definition and adds additional runtime defaults and extra arguments, based on OS environment variables
	EvalBuildArgs = func(mode config.RunMode, workspace string, dockerfilePath string, args map[string]*string, extra map[string]*string) (map[string]*string, error) {
		return evalBuildArgs(mode, workspace, dockerfilePath, args, extra, nil)
	}

	// default build args for skaffold non-debug mode
	nonDebugModeArgs = map[string]string{}
	// default build args for skaffold debug mode
	debugModeArgs = map[string]string{
		"SKAFFOLD_GO_GCFLAGS": "all=-N -l", // disable build optimization for Golang
		// TODO: Add for other languages
	}
)

// evalBuildArgs evaluates the build args provided in the artifact definition and adds other default and extra arguments, based on OS and custom environment variables.
// Use `EvalBuildArgs` or `EvalBuildArgsWithEnv` instead of using this directly.
func evalBuildArgs(mode config.RunMode, workspace string, dockerfilePath string, args map[string]*string, extra map[string]*string, env map[string]string) (map[string]*string, error) {
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

	for k, v := range extra {
		result[k] = v
	}

	absDockerfilePath, err := NormalizeDockerfilePath(workspace, dockerfilePath)
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
	for k, v := range args {
		result[k] = v
	}
	result, err = util.EvaluateEnvTemplateMapWithEnv(result, env)
	if err != nil {
		return nil, fmt.Errorf("unable to expand build args: %w", err)
	}
	return result, nil
}

// ArtifactResolver provides an interface to resolve built artifact tags by image name.
type ArtifactResolver interface {
	GetImageTag(imageName string) (string, bool)
}

// ResolveDependencyImages creates a map of artifact aliases to their built image from a required artifacts slice.
// If `missingIsFatal` is false then it is permissive of missing entries in the ArtifactResolver and returns nil for those entries.
func ResolveDependencyImages(deps []*latest.ArtifactDependency, r ArtifactResolver, missingIsFatal bool) map[string]*string {
	if r == nil {
		// `diagnose` is called without an artifact resolver. Return an empty map in this case.
		return nil
	}
	m := make(map[string]*string)
	for _, d := range deps {
		t, found := r.GetImageTag(d.ImageName)
		switch {
		case found:
			m[d.Alias] = &t
		case missingIsFatal:
			logrus.Fatalf("failed to resolve build result for required artifact %q", d.ImageName)
		default:
			m[d.Alias] = nil
		}
	}
	return m
}
