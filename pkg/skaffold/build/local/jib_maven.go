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

func (b *Builder) buildJibMaven(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, tag string) (string, error) {
	if b.pushImages {
		return b.buildJibMavenToRegistry(ctx, out, workspace, artifact, tag)
	}
	return b.buildJibMavenToDocker(ctx, out, workspace, artifact, tag)
}

func (b *Builder) buildJibMavenToDocker(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, tag string) (string, error) {
	args := jib.GenerateMavenArgs("dockerBuild", tag, artifact, b.skipTests, b.insecureRegistries)
	if err := b.runMavenCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return b.localDocker.ImageID(ctx, tag)
}

func (b *Builder) buildJibMavenToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibArtifact, tag string) (string, error) {
	args := jib.GenerateMavenArgs("build", tag, artifact, b.skipTests, b.insecureRegistries)
	if err := b.runMavenCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return docker.RemoteDigest(tag, b.insecureRegistries)
}

func (b *Builder) runMavenCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	cmd := jib.MavenCommand.CreateCommand(ctx, workspace, args)
	cmd.Env = append(util.OSEnviron(), b.localDocker.ExtraEnv()...)
	cmd.Stdout = out
	cmd.Stderr = out

	logrus.Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(&cmd); err != nil {
		return errors.Wrap(err, "maven build failed")
	}

	return nil
}
