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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
)

func (b *Builder) SupportedPlatforms() platform.Matcher {
	return platform.All
}

func (b *Builder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {
	var pl v1.Platform
	if len(matcher.Platforms) == 1 {
		pl = util.ConvertToV1Platform(matcher.Platforms[0])
	}
	a = adjustCacheFrom(a, tag)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "docker",
		"Context":     instrumentation.PII(a.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	// Fail fast if the Dockerfile can't be found.
	dockerfile, err := docker.NormalizeDockerfilePath(a.Workspace, a.DockerArtifact.DockerfilePath)
	if err != nil {
		return "", dockerfileNotFound(fmt.Errorf("normalizing dockerfile path: %w", err), a.ImageName)
	}
	if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
		return "", dockerfileNotFound(err, a.ImageName)
	}

	if err := b.pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact, pl); err != nil {
		return "", cacheFromPullErr(err, a.ImageName)
	}
	opts := docker.BuildOptions{Tag: tag, Mode: b.cfg.Mode(), ExtraBuildArgs: docker.ResolveDependencyImages(a.Dependencies, b.artifacts, true)}

	var imageID string

	// ignore useCLI boolean if buildkit is enabled since buildkit is only implemented for docker CLI at the moment in skaffold.
	// we might consider a different approach in the future.
	// use CLI for cross-platform builds
	if b.useCLI || b.buildx || (b.useBuildKit != nil && *b.useBuildKit) || len(a.DockerArtifact.CliFlags) > 0 || matcher.IsNotEmpty() {
		imageID, err = b.dockerCLIBuild(ctx, output.GetUnderlyingWriter(out), a.ImageName, a.Workspace, dockerfile, a.ArtifactType.DockerArtifact, opts, pl)
	} else {
		imageID, err = b.localDocker.Build(ctx, out, a.Workspace, a.ImageName, a.ArtifactType.DockerArtifact, opts)
	}

	if err != nil {
		return "", newBuildError(err, b.cfg)
	}

	if b.pushImages {
		// TODO (tejaldesai) Remove https://github.com/GoogleContainerTools/skaffold/blob/main/pkg/skaffold/errors/err_map.go#L56
		// and instead define a pushErr() method here.
		return b.localDocker.Push(ctx, out, tag)
	}

	return imageID, nil
}

func (b *Builder) dockerCLIBuild(ctx context.Context, out io.Writer, name string, workspace string, dockerfilePath string, a *latest.DockerArtifact, opts docker.BuildOptions, pl v1.Platform) (string, error) {
	args := []string{"build", workspace, "--file", dockerfilePath, "-t", opts.Tag}
	if b.buildx {
		args = append([]string{"buildx"}, args...)
	}
	imgRef, err := docker.ParseReference(opts.Tag)
	if err != nil {
		return "", fmt.Errorf("couldn't parse image tag: %w", err)
	}
	imageInfoEnv := map[string]string{
		"IMAGE_REPO": imgRef.Repo,
		"IMAGE_NAME": imgRef.Name,
		"IMAGE_TAG":  imgRef.Tag,
	}
	ba, err := docker.EvalBuildArgsWithEnv(b.cfg.Mode(), workspace, a.DockerfilePath, a.BuildArgs, opts.ExtraBuildArgs, imageInfoEnv)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate build args: %w", err)
	}
	cliArgs, err := docker.ToCLIBuildArgs(a, ba)
	if err != nil {
		return "", fmt.Errorf("getting docker build args: %w", err)
	}
	if b.buildx && b.pushImages {
		cliArgs = append(cliArgs, "--push")
	}
	args = append(args, cliArgs...)

	if b.cfg.Prune() {
		args = append(args, "--force-rm")
	}

	if pl.String() != "" {
		args = append(args, "--platform", pl.String())
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(util.OSEnviron(), b.localDocker.ExtraEnv()...)
	if b.useBuildKit != nil {
		if *b.useBuildKit {
			cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")
		} else {
			cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=0")
		}
	} else if pl.String() != "" { // cross-platform builds require buildkit
		log.Entry(ctx).Debugf("setting DOCKER_BUILDKIT=1 for docker build for artifact %q since it targets platform %q", name, pl.String())
		cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")
	}
	cmd.Stdout = out

	var errBuffer bytes.Buffer
	stderr := io.MultiWriter(out, &errBuffer)
	cmd.Stderr = stderr

	if err := util.RunCmd(ctx, cmd); err != nil {
		return "", tryExecFormatErr(fmt.Errorf("running build: %w", err), errBuffer)
	}

	if b.buildx {
		return "", nil // TODO: return id from CLI
	} else {
		return b.localDocker.ImageID(ctx, opts.Tag)
	}
}

func (b *Builder) pullCacheFromImages(ctx context.Context, out io.Writer, a *latest.DockerArtifact, pl v1.Platform) error {
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

		if err := b.localDocker.Pull(ctx, out, image, pl); err != nil {
			warnings.Printf("cacheFrom image %q couldn't be pulled for platform %q\n", image, pl)
		}
	}

	return nil
}

// adjustCacheFrom returns an artifact where any cache references from the artifactImage is changed to the tagged built image name instead.
func adjustCacheFrom(a *latest.Artifact, artifactTag string) *latest.Artifact {
	if os.Getenv("SKAFFOLD_DISABLE_DOCKER_CACHE_ADJUSTMENT") != "" {
		// allow this behaviour to be disabled
		return a
	}

	if !stringslice.Contains(a.DockerArtifact.CacheFrom, a.ImageName) {
		return a
	}

	cf := make([]string, 0, len(a.DockerArtifact.CacheFrom))
	for _, image := range a.DockerArtifact.CacheFrom {
		if image == a.ImageName {
			cf = append(cf, artifactTag)
		} else {
			cf = append(cf, image)
		}
	}
	copy := *a
	copy.DockerArtifact.CacheFrom = cf
	return &copy
}
