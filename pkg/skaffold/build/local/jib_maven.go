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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/pkg/errors"
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
