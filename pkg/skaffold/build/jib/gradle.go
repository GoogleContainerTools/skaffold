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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
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
const MinimumJibGradleVersionForSync = "2.0.0"

// GradleCommand stores Gradle executable and wrapper name
var GradleCommand = util.CommandWrapper{Executable: "gradle", Wrapper: "gradlew"}

func (b *Builder) buildJibGradleToDocker(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, deps []*latest.ArtifactDependency, tag string) (string, error) {
	args := GenerateGradleBuildArgs("jibDockerBuild", tag, artifact, b.skipTests, b.pushImages, deps, b.artifacts, b.cfg.GetInsecureRegistries(), color.IsColorable(out))
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", jibToolErr(err)
	}

	return b.localDocker.ImageID(ctx, tag)
}

func (b *Builder) buildJibGradleToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, deps []*latest.ArtifactDependency, tag string) (string, error) {
	args := GenerateGradleBuildArgs("jib", tag, artifact, b.skipTests, b.pushImages, deps, b.artifacts, b.cfg.GetInsecureRegistries(), color.IsColorable(out))
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", jibToolErr(err)
	}

	return docker.RemoteDigest(tag, b.cfg)
}

func (b *Builder) runGradleCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	cmd := GradleCommand.CreateCommand(ctx, workspace, args)
	cmd.Env = append(util.OSEnviron(), b.localDocker.ExtraEnv()...)
	cmd.Stdout = out
	cmd.Stderr = out

	logrus.Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(&cmd); err != nil {
		return fmt.Errorf("gradle build failed: %w", err)
	}

	return nil
}

// getDependenciesGradle finds the source dependencies for the given jib-gradle artifact.
// All paths are absolute.
func getDependenciesGradle(ctx context.Context, workspace string, a *latest.JibArtifact) ([]string, error) {
	cmd := getCommandGradle(ctx, workspace, a)
	deps, err := getDependencies(workspace, cmd, a)
	if err != nil {
		return nil, dependencyErr(JibGradle, workspace, err)
	}
	logrus.Debugf("Found dependencies for jib-gradle artifact: %v", deps)
	return deps, nil
}

func getCommandGradle(ctx context.Context, workspace string, a *latest.JibArtifact) exec.Cmd {
	args := append(gradleArgsFunc(a, "_jibSkaffoldFilesV2", MinimumJibGradleVersion), "-q", "--console=plain")
	return GradleCommand.CreateCommand(ctx, workspace, args)
}

func getSyncMapCommandGradle(ctx context.Context, workspace string, a *latest.JibArtifact) *exec.Cmd {
	cmd := GradleCommand.CreateCommand(ctx, workspace, gradleBuildArgsFunc("_jibSkaffoldSyncMap", a, true, false, MinimumJibMavenVersionForSync))
	return &cmd
}

// GenerateGradleBuildArgs generates the arguments to Gradle for building the project as an image.
func GenerateGradleBuildArgs(task string, imageName string, a *latest.JibArtifact, skipTests, pushImages bool, deps []*latest.ArtifactDependency, r ArtifactResolver, insecureRegistries map[string]bool, showColors bool) []string {
	args := gradleBuildArgsFunc(task, a, skipTests, showColors, MinimumJibGradleVersion)
	if insecure, err := isOnInsecureRegistry(imageName, insecureRegistries); err == nil && insecure {
		// jib doesn't support marking specific registries as insecure
		args = append(args, "-Djib.allowInsecureRegistries=true")
	}
	if baseImg, found := baseImageArg(a, r, deps, pushImages); found {
		args = append(args, baseImg)
	}
	args = append(args, "--image="+imageName)
	return args
}

// Do not use directly, use gradleBuildArgsFunc
func gradleBuildArgs(task string, a *latest.JibArtifact, skipTests, showColors bool, minimumVersion string) []string {
	// Disable jib's rich progress footer on builds. Show colors on normal builds for clearer information,
	// but use --console=plain for internal goals to avoid formatting issues
	var args []string
	if showColors {
		args = []string{"-Djib.console=plain"}
	} else {
		args = []string{"--console=plain"}
	}
	args = append(args, gradleArgsFunc(a, task, minimumVersion)...)

	if skipTests {
		args = append(args, "-x", "test")
	}
	args = append(args, a.Flags...)
	return args
}

// Do not use directly, use gradleArgsFunc
func gradleArgs(a *latest.JibArtifact, task string, minimumVersion string) []string {
	args := []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + minimumVersion}
	if a.Project == "" {
		return append(args, ":"+task)
	}

	// multi-module
	return append(args, fmt.Sprintf(":%s:%s", a.Project, task))
}
