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
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Skaffold-Jib depends on functionality introduced with Jib-Maven 1.4.0
const MinimumJibMavenVersion = "1.4.0"

// MavenCommand stores Maven executable and wrapper name
var MavenCommand = util.CommandWrapper{Executable: "mvn", Wrapper: "mvnw"}

// getDependenciesMaven finds the source dependencies for the given jib-maven artifact.
// All paths are absolute.
func getDependenciesMaven(ctx context.Context, workspace string, a *latest.JibArtifact) ([]string, error) {
	deps, err := getDependencies(workspace, getCommandMaven(ctx, workspace, a), a.Project)
	if err != nil {
		return nil, errors.Wrapf(err, "getting jib-maven dependencies")
	}
	logrus.Debugf("Found dependencies for jib maven artifact: %v", deps)
	return deps, nil
}

func getCommandMaven(ctx context.Context, workspace string, a *latest.JibArtifact) exec.Cmd {
	args := mavenArgs(a)
	args = append(args, "jib:_skaffold-files-v2", "--quiet")

	return MavenCommand.CreateCommand(ctx, workspace, args)
}

// GenerateMavenArgs generates the arguments to Maven for building the project as an image.
func GenerateMavenArgs(goal string, imageName string, a *latest.JibArtifact, skipTests bool, insecureRegistries map[string]bool) []string {
	// disable jib's rich progress footer on builds; we could use --batch-mode
	// but it also disables colour which can be helpful
	args := []string{"-Djib.console=plain"}
	args = append(args, mavenArgs(a)...)

	if skipTests {
		args = append(args, "-DskipTests=true")
	}

	if a.Project == "" {
		// single-module project
		args = append(args, "prepare-package", "jib:"+goal)
	} else {
		// multi-module project: instruct jib to containerize only the given module
		args = append(args, "package", "jib:"+goal, "-Djib.containerize="+a.Project)
	}

	if insecure, err := isOnInsecureRegistry(imageName, insecureRegistries); err == nil && insecure {
		// jib doesn't support marking specific registries as insecure
		args = append(args, "-Djib.allowInsecureRegistries=true")
	}
	args = append(args, "-Dimage="+imageName)

	return args
}

func mavenArgs(a *latest.JibArtifact) []string {
	args := []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=" + MinimumJibMavenVersion}
	args = append(args, a.Flags...)

	if a.Project == "" {
		// single-module project
		args = append(args, "--non-recursive")
	} else {
		// multi-module project
		args = append(args, "--projects", a.Project, "--also-make")
	}

	return args
}
