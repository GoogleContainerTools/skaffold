/*
Copyright YEAR The Skaffold Authors

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

package please

import (
	"context"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func GetDependencies(ctx context.Context, workspace string, a *latest.PleaseArtifact) ([]string, error) {
	deps, err := getInputFiles(ctx, workspace, a)
	if err != nil {
		return nil, errors.Wrap(err, "getting please input files dependencies")
	}

	targets, err := getDepTargets(ctx, workspace, a)

	if err != nil {
		return nil, errors.Wrap(err, "getting please dependencies")
	}

	deps = append(deps, targets...)

	// this is here only for please projects that follow bazel project structure
	if _, err := os.Stat(filepath.Join(workspace, "WORKSPACE")); err == nil {
		deps = append(deps, "WORKSPACE")
	}

	logrus.Debugf("Found dependencies for please artifact: %v", deps)

	return deps, nil
}

func getDepTargets(ctx context.Context, workspace string, a *latest.PleaseArtifact) ([]string, error) {
	var deps []string
	cmd := exec.CommandContext(ctx, "please", "query", "deps", "-p", "-u", a.BuildTarget)
	cmd.Dir = workspace
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "getting please dependencies")
	}

	labels := strings.Split(string(stdout), "\n")
	for _, l := range labels {
		if l != "" {
			d := path.Join(depToPath(l), "BUILD")
			if !containsPath(deps, d) {
				deps = append(deps, d)
			}
		}
	}
	return deps, nil
}

func getInputFiles(ctx context.Context, workspace string, a *latest.PleaseArtifact) ([]string, error) {
	cmd := exec.CommandContext(ctx, "please", "query", "input", a.BuildTarget)
	cmd.Dir = workspace
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "getting please input files dependencies")
	}

	labels := strings.Split(string(stdout), "\n")
	var deps []string
	for _, l := range labels {
		if l != "" {
			deps = append(deps, l)
		}
	}
	return deps, nil
}

func containsPath(deps []string, elm string) bool {
	for _, e := range deps {
		if e == elm {
			return true
		}
	}
	return false
}

func depToPath(dep string) string {
	return strings.Split(strings.TrimPrefix(dep, "//"), ":")[0]
}
