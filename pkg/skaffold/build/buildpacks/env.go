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

package buildpacks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/pack/project"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func GetEnv(a *latest.Artifact, mode config.RunMode) (map[string]string, error) {
	artifact := a.BuildpackArtifact
	workspace := a.Workspace

	path := filepath.Join(workspace, artifact.ProjectDescriptor)
	projectDescriptor, err := project.ReadProjectDescriptor(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read project descriptor %q: %w", path, err)
	}
	return env(a, mode, projectDescriptor)
}

func env(a *latest.Artifact, mode config.RunMode, projectDescriptor project.Descriptor) (map[string]string, error) {
	envVars, err := misc.EvaluateEnv(a.BuildpackArtifact.Env)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate env variables: %w", err)
	}

	if mode == config.RunModes.Dev && a.Sync != nil && a.Sync.Auto != nil && *a.Sync.Auto {
		envVars = append(envVars, "GOOGLE_DEVMODE=1")
	}

	env := envMap(envVars)
	for _, kv := range projectDescriptor.Build.Env {
		env[kv.Name] = kv.Value
	}
	env = addDefaultArgs(mode, env)
	return env, nil
}
