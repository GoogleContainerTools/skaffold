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
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

func (b *Builder) buildJibGradle(ctx context.Context, out io.Writer, workspace string, a *latest.JibGradleArtifact) (string, error) {
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
func generateGradleCommand(_ /*workspace*/ string, skaffoldImage string, a *latest.JibGradleArtifact) []string {
	var command []string
	if a.Project == "" {
		command = []string{":jibDockerBuild"}
	} else {
		// multi-module
		command = []string{fmt.Sprintf(":%s:jibDockerBuild", a.Project)}
	}
	command = append(command, "--image="+skaffoldImage)
	return command
}
