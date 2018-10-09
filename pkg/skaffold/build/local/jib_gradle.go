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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Builder) buildJibGradleToDocker(ctx context.Context, out io.Writer, workspace string, a *latest.JibGradleArtifact) (string, error) {
	skaffoldImage := generateJibImageRef(workspace, a.Project)
	args := generateGradleArgs("jibDockerBuild", skaffoldImage, a)

	if err := runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return skaffoldImage, nil
}

func (b *Builder) buildJibGradleToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.Artifact) (string, error) {
	initialTag := util.RandomID()
	skaffoldImage := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)
	args := generateGradleArgs("jib", skaffoldImage, artifact.JibGradleArtifact)

	if err := runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return skaffoldImage, nil
}

// generateGradleArgs generates the arguments to Gradle for building the project as an image called `skaffoldImage`.
func generateGradleArgs(task string, skaffoldImage string, a *latest.JibGradleArtifact) []string {
	var command string
	if a.Project == "" {
		command = ":" + task
	} else {
		// multi-module
		command = fmt.Sprintf(":%s:%s", a.Project, task)
	}

	return []string{command, "--image=" + skaffoldImage}
}

func runGradleCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	cmd := jib.GradleCommand.CreateCommand(ctx, workspace, args)
	cmd.Stdout = out
	cmd.Stderr = out

	logrus.Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(cmd); err != nil {
		return errors.Wrap(err, "gradle build failed")
	}

	return nil
}
