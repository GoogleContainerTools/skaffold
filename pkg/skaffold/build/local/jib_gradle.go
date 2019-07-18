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

package local

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Builder) buildJibGradle(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibGradleArtifact, tag string) (string, error) {
	if b.pushImages {
		return b.buildJibGradleToRegistry(ctx, out, workspace, artifact, tag)
	}
	return b.buildJibGradleToDocker(ctx, out, workspace, artifact, tag)
}

func (b *Builder) buildJibGradleToDocker(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibGradleArtifact, tag string) (string, error) {
	args := jib.GenerateGradleArgs("jibDockerBuild", tag, artifact, b.skipTests)
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return b.localDocker.ImageID(ctx, tag)
}

func (b *Builder) buildJibGradleToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibGradleArtifact, tag string) (string, error) {
	args := jib.GenerateGradleArgs("jib", tag, artifact, b.skipTests)
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return docker.RemoteDigest(tag, b.insecureRegistries)
}

func (b *Builder) runGradleCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	cmd := jib.GradleCommand.CreateCommand(ctx, workspace, args)
	cmd.Env = append(util.OSEnviron(), b.localDocker.ExtraEnv()...)
	cmd.Stdout = out
	cmd.Stderr = out

	logrus.Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(&cmd); err != nil {
		return errors.Wrap(err, "gradle build failed")
	}

	return nil
}
