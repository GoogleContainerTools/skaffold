/*
Copyright 2020 The Skaffold Authors

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

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

func (b *Builder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	// Fail fast if the Dockerfile can't be found.
	dockerfile, err := docker.NormalizeDockerfilePath(a.Workspace, a.DockerArtifact.DockerfilePath)
	if err != nil {
		return "", dockerfileNotFound(fmt.Errorf("normalizing dockerfile path: %w", err), a.ImageName)
	}
	if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
		return "", dockerfileNotFound(err, a.ImageName)
	}

	if err := b.pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact); err != nil {
		return "", cacheFromPullErr(err, a.ImageName)
	}
	opts := docker.BuildOptions{Tag: tag, Mode: b.mode, ExtraBuildArgs: docker.ResolveDependencyImages(a.Dependencies, b.artifacts, true)}

	var imageID string

	if b.useCLI || b.useBuildKit {
		imageID, err = b.dockerCLIBuild(ctx, color.GetWriter(out), a.Workspace, dockerfile, a.ArtifactType.DockerArtifact, opts)
	} else {
		imageID, err = b.localDocker.Build(ctx, out, a.Workspace, a.ImageName, a.ArtifactType.DockerArtifact, opts)
	}

	if err != nil {
		return "", newBuildError(err)
	}

	if b.pushImages {
		// TODO (tejaldesai) Remove https://github.com/GoogleContainerTools/skaffold/blob/master/pkg/skaffold/errors/err_map.go#L56
		// and instead define a pushErr() method here.
		return b.localDocker.Push(ctx, out, tag)
	}

	return imageID, nil
}

func (b *Builder) dockerCLIBuild(ctx context.Context, out io.Writer, workspace string, dockerfilePath string, a *latest.DockerArtifact, opts docker.BuildOptions) (string, error) {
	args := []string{"build", workspace, "--file", dockerfilePath, "-t", opts.Tag}
	ba, err := docker.EvalBuildArgs(b.mode, workspace, a.DockerfilePath, a.BuildArgs, opts.ExtraBuildArgs)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate build args: %w", err)
	}
	cliArgs, err := docker.ToCLIBuildArgs(a, ba)
	if err != nil {
		return "", fmt.Errorf("getting docker build args: %w", err)
	}
	args = append(args, cliArgs...)

	if b.prune {
		args = append(args, "--force-rm")
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(util.OSEnviron(), b.localDocker.ExtraEnv()...)
	if b.useBuildKit {
		cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")
	}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := util.RunCmd(cmd); err != nil {
		return "", fmt.Errorf("running build: %w", err)
	}

	return b.localDocker.ImageID(ctx, opts.Tag)
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
			warnings.Printf("cacheFrom image couldn't be pulled: %s\n", image)
		}
	}

	return nil
}
