/*
Copyright 2018 The Skaffold Authors

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

package jib

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var GradleCommand = util.CommandWrapper{Executable: "gradle", Wrapper: "gradlew"}

// GetDependenciesGradle finds the source dependencies for the given jib-gradle artifact.
// All paths are absolute.
func GetDependenciesGradle(ctx context.Context, workspace string, a *latest.JibGradleArtifact) ([]string, error) {
	cmd := getCommandGradle(ctx, workspace, a)
	deps, err := getDependencies(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "getting jibGradle dependencies")
	}
	logrus.Debugf("Found dependencies for jibGradle artifact: %v", deps)
	return deps, nil
}

func getCommandGradle(ctx context.Context, workspace string, a *latest.JibGradleArtifact) *exec.Cmd {
	args := []string{gradleCommand(a, "_jibSkaffoldFiles"), "-q"}
	return GradleCommand.CreateCommand(ctx, workspace, args)
}

// GenerateGradleArgs generates the arguments to Gradle for building the project as an image.
func GenerateGradleArgs(task string, imageName string, a *latest.JibGradleArtifact) []string {
	return []string{gradleCommand(a, task), "--image=" + imageName}
}

func gradleCommand(a *latest.JibGradleArtifact, task string) string {
	if a.Project == "" {
		return ":" + task
	}

	// multi-module
	return fmt.Sprintf(":%s:%s", a.Project, task)
}
