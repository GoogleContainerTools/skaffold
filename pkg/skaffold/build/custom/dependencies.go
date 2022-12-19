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

package custom

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/list"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// GetDependencies returns dependencies listed for a custom artifact
func GetDependencies(ctx context.Context, workspace string, artifactName string, a *latest.CustomArtifact, cfg docker.Config) ([]string, error) {
	switch {
	case a.Dependencies.Dockerfile != nil:
		return docker.GetDependencies(ctx, getDockerBuildConfig(workspace, artifactName, a), cfg)

	case a.Dependencies.Command != "":
		split := strings.Split(a.Dependencies.Command, " ")
		cmd := exec.CommandContext(ctx, split[0], split[1:]...)
		cmd.Dir = workspace
		output, err := util.RunCmdOut(ctx, cmd)
		if err != nil {
			return nil, fmt.Errorf("getting dependencies from command: %q: %w", a.Dependencies.Command, err)
		}
		var deps []string
		if err := json.Unmarshal(output, &deps); err != nil {
			return nil, fmt.Errorf("unmarshalling dependency output into string array: %w", err)
		}
		return deps, nil

	default:
		return list.Files(workspace, a.Dependencies.Paths, a.Dependencies.Ignore)
	}
}

func getDockerBuildConfig(ws string, artifact string, a *latest.CustomArtifact) docker.BuildConfig {
	dockerfile := a.Dependencies.Dockerfile
	return docker.NewBuildConfig(ws, artifact, dockerfile.Path, dockerfile.BuildArgs)
}
