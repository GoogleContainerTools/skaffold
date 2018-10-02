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

package local

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os/exec"
	"regexp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Builder) buildJibMaven(ctx context.Context, out io.Writer, workspace string, a *v1alpha3.JibMavenArtifact) (string, error) {
	skaffoldImage := generateJibImageRef(workspace, a.Module)

	maven, err := findBuilder("mvn", "mvnw", workspace)
	if err != nil {
		return "", errors.Wrap(err, "Unable to find maven executable")
	}
	mavenCommand, err := generateMavenCommand(workspace, skaffoldImage, a)
	if err != nil {
		return "", err
	}
	commandLine := append(maven, mavenCommand...)

	err = executeBuildCommand(ctx, out, workspace, commandLine)
	if err != nil {
		return "", errors.Wrap(err, "maven build failed")
	}
	return skaffoldImage, nil
}

// generateMavenCommand generates the command-line to pass to maven for building a
// project found in `workspace`.  The resulting image is added to the local docker daemon
// and called `skaffoldImage`.
func generateMavenCommand(_ /*workspace*/ string, skaffoldImage string, a *v1alpha3.JibMavenArtifact) ([]string, error) {
	if a.Module != "" {
		// TODO: multi-module
		return nil, errors.New("Maven multi-modules not supported yet")
	}
	// use mostly-qualified plugin ID in case jib is not a configured plugin
	commandLine := []string{"prepare-package", "com.google.cloud.tools:jib-maven-plugin::dockerBuild", "-Dimage=" + skaffoldImage}
	if a.Profile != "" {
		commandLine = append(commandLine, "-P"+a.Profile)
	}
	return commandLine, nil
}

func (b *Builder) buildJibGradle(ctx context.Context, out io.Writer, workspace string, a *v1alpha3.JibGradleArtifact) (string, error) {
	skaffoldImage := generateJibImageRef(workspace, a.Project)
	gradle, err := findBuilder("gradle", "gradlew", workspace)
	if err != nil {
		return "", errors.Wrap(err, "Unable to find gradle executable")
	}
	gradleCommand := generateGradleCommand(workspace, skaffoldImage, a)
	commandLine := append(gradle, gradleCommand...)

	err = executeBuildCommand(ctx, out, workspace, commandLine)
	if err != nil {
		return "", errors.Wrap(err, "gradle build failed")
	}
	return skaffoldImage, nil
}

// generateGradleCommand generates the command-line to pass to gradle for building an
// project in `workspace`.  The resulting image is added to the local docker daemon
// and called `skaffoldImage`.
func generateGradleCommand(_ /*workspace*/ string, skaffoldImage string, a *v1alpha3.JibGradleArtifact) []string {
	command := []string{}
	if a.Project == "" {
		command = append(command, ":jibDockerBuild")
	} else {
		// multi-module
		command = append(command, ":"+a.Project+":jibDockerBuild")
	}
	command = append(command, "--image="+skaffoldImage)
	return command
}

// executeBuildCommand executes the command-line with the working directory set to `workspace`.
func executeBuildCommand(ctx context.Context, out io.Writer, workspace string, commandLine []string) error {
	logrus.Infof("Building %v: %v", workspace, commandLine)
	cmd := exec.CommandContext(ctx, commandLine[0], commandLine[1:]...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	return util.RunCmd(cmd)
}

// jibBuildImageRef generates a valid image name for the workspace and project.
// The image name is always prefixed with `jib`.
func generateJibImageRef(workspace string, project string) string {
	imageName := "jib" + workspace
	if project != "" {
		imageName += "_" + project
	}
	// if the workspace + project is a valid image name then use it
	match := regexp.MustCompile(constants.RepositoryComponentRegex).MatchString(imageName)
	if match {
		return imageName
	}
	// otherwise use a hash for a deterministic name
	hasher := sha1.New()
	io.WriteString(hasher, imageName)
	return "jib__" + hex.EncodeToString(hasher.Sum(nil))
}
