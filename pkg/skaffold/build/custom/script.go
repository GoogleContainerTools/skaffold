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

package custom

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	// For testing
	buildContext = retrieveBuildContext
)

func (b *Builder) runBuildScript(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) error {
	cmd, err := b.retrieveCmd(ctx, out, a, tag)
	if err != nil {
		return fmt.Errorf("retrieving cmd: %w", err)
	}

	logrus.Debugf("Running command: %s", cmd.Args)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting cmd: %w", err)
	}

	return misc.HandleGracefulTermination(ctx, cmd)
}

func (b *Builder) retrieveCmd(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (*exec.Cmd, error) {
	artifact := a.CustomArtifact

	// Expand command
	command, err := util.ExpandEnvTemplate(artifact.BuildCommand, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to parse build command %q: %w", artifact.BuildCommand, err)
	}

	var cmd *exec.Cmd
	// We evaluate the command with a shell so that it can contain
	// env variables.
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Stdout = out
	cmd.Stderr = out

	env, err := b.retrieveEnv(a, tag)
	if err != nil {
		return nil, fmt.Errorf("retrieving env variables for %q: %w", a.ImageName, err)
	}
	cmd.Env = env

	dir, err := buildContext(a.Workspace)
	if err != nil {
		return nil, fmt.Errorf("getting context for artifact: %w", err)
	}
	cmd.Dir = dir

	return cmd, nil
}

func (b *Builder) retrieveEnv(a *latest.Artifact, tag string) ([]string, error) {
	buildContext, err := buildContext(a.Workspace)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path for artifact build context: %w", err)
	}

	envs := []string{
		fmt.Sprintf("%s=%s", constants.Image, tag),
		fmt.Sprintf("%s=%s", constants.DeprecatedImages, tag),
		fmt.Sprintf("%s=%t", constants.PushImage, b.pushImages),
		fmt.Sprintf("%s=%s", constants.BuildContext, buildContext),
	}

	ref, err := docker.ParseReference(tag)
	if err != nil {
		return nil, fmt.Errorf("parsing image %v: %w", tag, err)
	}

	// Standardize access to Image reference fields in templates
	envs = append(envs, fmt.Sprintf("%s=%s", constants.ImageRef.Repo, ref.BaseName))
	envs = append(envs, fmt.Sprintf("%s=%s", constants.ImageRef.Tag, ref.Tag))

	envs = append(envs, b.additionalEnv...)
	envs = append(envs, util.OSEnviron()...)
	return envs, nil
}

func retrieveBuildContext(workspace string) (string, error) {
	return filepath.Abs(workspace)
}
