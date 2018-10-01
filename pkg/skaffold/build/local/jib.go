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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Builder) buildJibMaven(ctx context.Context, out io.Writer, workspace string, a *v1alpha3.JibMavenArtifact) (string, error) {
	skaffoldImage := generateJibImageRef(workspace, a.Module)

	// TODO: multi-module
	// use mostly-qualified plugin ID in case jib is not a configured plugin
	commandLine, err := buildMavenCommand(workspace, "prepare-package", "com.google.cloud.tools:jib-maven-plugin::dockerBuild", "-Dimage="+skaffoldImage)
	if err != nil {
		return "", err
	}
	if a.Profile != "" {
		commandLine = append(commandLine, "-P"+a.Profile)
	}
	logrus.Infof("Building %s: %s", workspace, commandLine)
	cmd := exec.CommandContext(ctx, commandLine[0], commandLine[1:]...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	return skaffoldImage, nil
}

func (b *Builder) buildJibGradle(ctx context.Context, out io.Writer, workspace string, a *v1alpha3.JibGradleArtifact) (string, error) {
	skaffoldImage := generateJibImageRef(workspace, a.Project)

	// TODO: multi-module
	commandLine, err := buildGradleCommand(workspace, ":jibDockerBuild", "--image="+skaffoldImage)
	if err != nil {
		return "", err
	}
	logrus.Infof("Building %s: %s", workspace, commandLine)
	cmd := exec.CommandContext(ctx, commandLine[0], commandLine[1:]...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return "", errors.Wrap(err, "running command")
	} 

	return skaffoldImage, nil
}

const (
	// regexp matching valid image names
	REPOSITORY_COMPONENT_REGEX string = `^[a-z\d]+(?:(?:[_.]|__|-+)[a-z\d]+)*$`
)

// jibBuildImageRef generates a valid image name for the workspace and project.
// The image name is always prefixed with `jib`.
func generateJibImageRef(workspace string, project string) string {
	imageName := "jib" + workspace
	if project != "" {
		imageName += "_" + project
	}
	// if the workspace + project is a valid image name then use it
	match, _ := regexp.MatchString(REPOSITORY_COMPONENT_REGEX, imageName);
	if match {
		return imageName
	}
	// otherwise use a hash for a deterministic name
	hasher := sha1.New()
	io.WriteString(hasher, imageName)
    return "jib__" + hex.EncodeToString(hasher.Sum(nil))
}
