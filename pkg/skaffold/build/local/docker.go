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
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

func (b *Builder) buildDocker(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	// Fail fast if the Dockerfile can't be found.
	dockerfile, err := docker.NormalizeDockerfilePath(a.Workspace, a.DockerArtifact.DockerfilePath)
	if err != nil {
		return "", fmt.Errorf("normalizing dockerfile path: %w", err)
	}
	if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
		return "", fmt.Errorf("dockerfile %q not found", dockerfile)
	}

	if err := b.pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact); err != nil {
		return "", fmt.Errorf("pulling cache-from images: %w", err)
	}

	var imageID string

	if b.cfg.UseDockerCLI || b.cfg.UseBuildkit {
		imageID, err = b.dockerCLIBuild(ctx, out, a.Workspace, a.ArtifactType.DockerArtifact, tag)
	} else {
		imageID, err = b.localDocker.Build(ctx, out, a.Workspace, a.ArtifactType.DockerArtifact, tag)
	}

	if err != nil {
		return "", err
	}

	if b.pushImages {
		return b.localDocker.Push(ctx, out, tag)
	}

	return imageID, nil
}

func (b *Builder) retrieveExtraEnv() []string {
	return b.localDocker.ExtraEnv()
}

func (b *Builder) dockerCLIBuild(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, tag string) (string, error) {
	dockerfilePath, err := docker.NormalizeDockerfilePath(workspace, a.DockerfilePath)
	if err != nil {
		return "", fmt.Errorf("normalizing dockerfile path: %w", err)
	}

	args := []string{"build", workspace, "--file", dockerfilePath, "-t", tag}
	ba, err := docker.GetBuildArgs(a)
	if err != nil {
		return "", fmt.Errorf("getting docker build args: %w", err)
	}
	args = append(args, ba...)

	if b.prune {
		args = append(args, "--force-rm")
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(util.OSEnviron(), b.retrieveExtraEnv()...)
	if b.cfg.UseBuildkit {
		cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")
	}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := util.RunCmd(cmd); err != nil {
		return "", fmt.Errorf("running build: %w", err)
	}

	return b.localDocker.ImageID(ctx, tag)
}

func (b *Builder) pullCacheFromImages(ctx context.Context, out io.Writer, a *latest.DockerArtifact) error {
	if len(a.CacheFrom) == 0 {
		return nil
	}

	for _, image := range a.CacheFrom {
		imageID, err := b.localDocker.ImageID(ctx, image)
		if err != nil {
			return fmt.Errorf("getting imageID for %q: %w", image, err)
		}
		if imageID != "" {
			// already pulled
			continue
		}

		if err := b.localDocker.Pull(ctx, out, image); err != nil {
			warnings.Printf("Cache-From image couldn't be pulled: %s\n", image)
		}
	}

	return nil
}
