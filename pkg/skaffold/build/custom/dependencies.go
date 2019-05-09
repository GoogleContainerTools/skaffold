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
	"os/exec"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// GetDependencies returns dependencies listed for a custom artifact
func GetDependencies(ctx context.Context, workspace string, a *latest.CustomArtifact, insecureRegistries map[string]bool) ([]string, error) {

	switch {
	case a.Dependencies.Dockerfile != nil:
		dockerfile := a.Dependencies.Dockerfile
		return docker.GetDependencies(ctx, workspace, dockerfile.Path, dockerfile.BuildArgs, insecureRegistries)

	case a.Dependencies.Command != "":
		split := strings.Split(a.Dependencies.Command, " ")
		cmd := exec.CommandContext(ctx, split[0], split[1:]...)
		output, err := util.RunCmdOut(cmd)
		if err != nil {
			return nil, errors.Wrapf(err, "getting dependencies from command: %s", a.Dependencies.Command)
		}
		var deps []string
		if err := json.Unmarshal(output, &deps); err != nil {
			return nil, errors.Wrap(err, "unmarshalling dependency output into string array")
		}
		return deps, nil

	default:
		files, err := docker.WalkWorkspace(workspace, a.Dependencies.Ignore, a.Dependencies.Paths)
		if err != nil {
			return nil, errors.Wrapf(err, "walking workspace %s", workspace)
		}
		var dependencies []string
		for file := range files {
			dependencies = append(dependencies, file)
		}
		sort.Strings(dependencies)
		return dependencies, nil
	}

}
