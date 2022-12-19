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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// For testing
var (
	mavenArgsFunc      = mavenArgs
	mavenBuildArgsFunc = mavenBuildArgs
)

// Skaffold-Jib depends on functionality introduced with Jib-Maven 1.4.0
const MinimumJibMavenVersion = "1.4.0"
const MinimumJibMavenVersionForSync = "2.0.0"
const MinimumJibMavenVersionForCrossPlatform = "3.2.0"

// MavenCommand stores Maven executable and wrapper name
var MavenCommand = util.CommandWrapper{Executable: "mvn", Wrapper: "mvnw"}

func (b *Builder) buildJibMavenToDocker(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, deps []*latest.ArtifactDependency, tag string, platforms platform.Matcher) (string, error) {
	args := GenerateMavenBuildArgs("dockerBuild", tag, artifact, platforms, b.skipTests, b.pushImages, deps, b.artifacts, b.cfg.GetInsecureRegistries(), output.IsColorable(out))
	if err := b.runMavenCommand(ctx, out, workspace, args); err != nil {
		return "", jibToolErr(err)
	}

	return b.localDocker.ImageID(ctx, tag)
}

func (b *Builder) buildJibMavenToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, deps []*latest.ArtifactDependency, tag string, platforms platform.Matcher) (string, error) {
	args := GenerateMavenBuildArgs("build", tag, artifact, platforms, b.skipTests, b.pushImages, deps, b.artifacts, b.cfg.GetInsecureRegistries(), output.IsColorable(out))
	if err := b.runMavenCommand(ctx, out, workspace, args); err != nil {
		return "", jibToolErr(err)
	}

	return docker.RemoteDigest(tag, b.cfg, nil)
}

func (b *Builder) runMavenCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	cmd := MavenCommand.CreateCommand(ctx, workspace, args)
	cmd.Env = append(util.OSEnviron(), b.localDocker.ExtraEnv()...)
	cmd.Stdout = out
	cmd.Stderr = out

	log.Entry(ctx).Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(ctx, &cmd); err != nil {
		return fmt.Errorf("maven build failed: %w", err)
	}

	return nil
}

// getDependenciesMaven finds the source dependencies for the given jib-maven artifact.
// All paths are absolute.
func getDependenciesMaven(ctx context.Context, workspace string, a *latest.JibArtifact) ([]string, error) {
	deps, err := getDependencies(ctx, workspace, getCommandMaven(ctx, workspace, a), a)
	if err != nil {
		return nil, dependencyErr(JibMaven, workspace, err)
	}
	log.Entry(ctx).Debugf("Found dependencies for jib maven artifact: %v", deps)
	return deps, nil
}

func getCommandMaven(ctx context.Context, workspace string, a *latest.JibArtifact) exec.Cmd {
	args := mavenArgsFunc(a, MinimumJibMavenVersion)
	args = append(args, "jib:_skaffold-files-v2", "--quiet", "--batch-mode")

	return MavenCommand.CreateCommand(ctx, workspace, args)
}

func getSyncMapCommandMaven(ctx context.Context, workspace string, a *latest.JibArtifact) *exec.Cmd {
	cmd := MavenCommand.CreateCommand(ctx, workspace, mavenBuildArgsFunc("_skaffold-sync-map", a, true, false, MinimumJibMavenVersionForSync))
	return &cmd
}

// GenerateMavenBuildArgs generates the arguments to Maven for building the project as an image.
func GenerateMavenBuildArgs(goal, imageName string, a *latest.JibArtifact, platforms platform.Matcher, skipTests, pushImages bool, deps []*latest.ArtifactDependency, r ArtifactResolver, insecureRegistries map[string]bool, showColors bool) []string {
	minVersion := MinimumJibMavenVersion
	if platforms.IsCrossPlatform() {
		minVersion = MinimumJibMavenVersionForCrossPlatform
	}
	args := mavenBuildArgsFunc(goal, a, skipTests, showColors, minVersion)
	if insecure, err := isOnInsecureRegistry(imageName, insecureRegistries); err == nil && insecure {
		// jib doesn't support marking specific registries as insecure
		args = append(args, "-Djib.allowInsecureRegistries=true")
	}
	if baseImg, found := baseImageArg(a, r, deps, pushImages); found {
		args = append(args, baseImg)
	}
	if platforms.IsNotEmpty() {
		args = append(args, fmt.Sprintf("-Djib.from.platforms=%s", platforms.String()))
	}
	args = append(args, "-Dimage="+imageName)
	return args
}

// Do not use directly, use mavenBuildArgsFunc
func mavenBuildArgs(goal string, a *latest.JibArtifact, skipTests, showColors bool, minimumVersion string) []string {
	// Disable jib's rich progress footer on builds. Show colors on normal builds for clearer information,
	// but use --batch-mode for internal goals to avoid formatting issues
	var args []string
	if showColors {
		args = []string{"-Dstyle.color=always", "-Djansi.passthrough=true", "-Djib.console=plain"}
	} else {
		args = []string{"--batch-mode"}
	}
	args = append(args, mavenArgsFunc(a, minimumVersion)...)

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
	return args
}

// Do not use directly, use mavenArgsFunc
func mavenArgs(a *latest.JibArtifact, minimumVersion string) []string {
	args := []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=" + minimumVersion}
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
