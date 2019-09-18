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

// Skaffold-Jib depends on functionality introduced with Jib-Gradle 1.4.0
const MinimumJibGradleVersion = "1.4.0"

// GradleCommand stores Gradle executable and wrapper name
var GradleCommand = util.CommandWrapper{Executable: "gradle", Wrapper: "gradlew"}

// getDependenciesGradle finds the source dependencies for the given jib-gradle artifact.
// All paths are absolute.
func getDependenciesGradle(ctx context.Context, workspace string, a *latest.JibArtifact) ([]string, error) {
	cmd := getCommandGradle(ctx, workspace, a)
	deps, err := getDependencies(workspace, cmd, a.Project)
	if err != nil {
		return nil, errors.Wrapf(err, "getting jib-gradle dependencies")
	}
	logrus.Debugf("Found dependencies for jib-gradle artifact: %v", deps)
	return deps, nil
}

func getCommandGradle(ctx context.Context, workspace string, a *latest.JibArtifact) exec.Cmd {
	args := append(gradleCommand(a, "_jibSkaffoldFilesV2"), "-q")
	return GradleCommand.CreateCommand(ctx, workspace, args)
}

// GenerateGradleArgs generates the arguments to Gradle for building the project as an image.
func GenerateGradleArgs(task string, imageName string, a *latest.JibArtifact, skipTests bool, insecureRegistries map[string]bool) []string {
	// disable jib's rich progress footer; we could use `--console=plain`
	// but it also disables colour which can be helpful
	args := []string{"-Djib.console=plain"}
	args = append(args, gradleCommand(a, task)...)

	if insecure, err := isOnInsecureRegistry(imageName, insecureRegistries); err == nil && insecure {
		// jib doesn't support marking specific registries as insecure
		args = append(args, "-Djib.allowInsecureRegistries=true")
	}

	args = append(args, "--image="+imageName)
	if skipTests {
		args = append(args, "-x", "test")
	}
	args = append(args, a.Flags...)
	return args
}

func gradleCommand(a *latest.JibArtifact, task string) []string {
	args := []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion}
	if a.Project == "" {
		return append(args, ":"+task)
	}

	// multi-module
	return append(args, fmt.Sprintf(":%s:%s", a.Project, task))
}
