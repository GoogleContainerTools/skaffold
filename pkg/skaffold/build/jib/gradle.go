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
	"io"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// For testing
var (
	gradleArgsFunc      = gradleArgs
	gradleBuildArgsFunc = gradleBuildArgs
)

// Skaffold-Jib depends on functionality introduced with Jib-Gradle 1.4.0
const MinimumJibGradleVersion = "1.4.0"

// GradleCommand stores Gradle executable and wrapper name
var GradleCommand = util.CommandWrapper{Executable: "gradle", Wrapper: "gradlew"}

func (b *Builder) buildJibGradleToDocker(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, tag string) (string, error) {
	args := GenerateGradleBuildArgs("jibDockerBuild", tag, artifact, b.skipTests, b.docker.InsecureRegistries())
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return b.docker.ImageID(ctx, tag)
}

func (b *Builder) buildJibGradleToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, tag string) (string, error) {
	args := GenerateGradleBuildArgs("jib", tag, artifact, b.skipTests, b.docker.InsecureRegistries())
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return b.docker.RemoteDigest(tag)
}

func (b *Builder) runGradleCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	extraEnv, err := b.docker.ExtraEnv()
	if err != nil {
		return err
	}

	cmd := GradleCommand.CreateCommand(ctx, workspace, args)
	cmd.Env = append(util.OSEnviron(), extraEnv...)
	cmd.Stdout = out
	cmd.Stderr = out

	logrus.Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(&cmd); err != nil {
		return errors.Wrap(err, "gradle build failed")
	}

	return nil
}

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
	args := append(gradleArgsFunc(a, "_jibSkaffoldFilesV2"), "-q")
	return GradleCommand.CreateCommand(ctx, workspace, args)
}

func getSyncMapCommandGradle(ctx context.Context, workspace string, a *latest.JibArtifact) *exec.Cmd {
	cmd := GradleCommand.CreateCommand(ctx, workspace, gradleBuildArgsFunc("_jibSkaffoldSyncMap", a, true))
	return &cmd
}

// GenerateGradleBuildArgs generates the arguments to Gradle for building the project as an image.
func GenerateGradleBuildArgs(task string, imageName string, a *latest.JibArtifact, skipTests bool, insecureRegistries map[string]bool) []string {
	args := gradleBuildArgsFunc(task, a, skipTests)
	if insecure, err := isOnInsecureRegistry(imageName, insecureRegistries); err == nil && insecure {
		// jib doesn't support marking specific registries as insecure
		args = append(args, "-Djib.allowInsecureRegistries=true")
	}

	args = append(args, "--image="+imageName)
	return args
}

// Do not use directly, use gradleBuildArgsFunc
func gradleBuildArgs(task string, a *latest.JibArtifact, skipTests bool) []string {
	// disable jib's rich progress footer; we could use `--console=plain`
	// but it also disables colour which can be helpful
	args := []string{"-Djib.console=plain"}
	args = append(args, gradleArgsFunc(a, task)...)

	if skipTests {
		args = append(args, "-x", "test")
	}
	args = append(args, a.Flags...)
	return args
}

// Do not use directly, use gradleArgsFunc
func gradleArgs(a *latest.JibArtifact, task string) []string {
	args := []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion}
	if a.Project == "" {
		return append(args, ":"+task)
	}

	// multi-module
	return append(args, fmt.Sprintf(":%s:%s", a.Project, task))
}
