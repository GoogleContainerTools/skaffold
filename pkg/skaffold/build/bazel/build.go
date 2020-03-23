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

package bazel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Build builds an artifact with Bazel.
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	a := artifact.ArtifactType.BazelArtifact

	tarPath, err := b.buildTar(ctx, out, artifact.Workspace, a)
	if err != nil {
		return "", err
	}

	if b.pushImages {
		return docker.Push(tarPath, tag, b.insecureRegistries)
	}
	return b.loadImage(ctx, out, tarPath, a, tag)
}

func (b *Builder) buildTar(ctx context.Context, out io.Writer, workspace string, a *latest.BazelArtifact) (string, error) {
	if !strings.HasSuffix(a.BuildTarget, ".tar") {
		return "", errors.New("the bazel build target should end with .tar, see https://github.com/bazelbuild/rules_docker#using-with-docker-locally")
	}

	args := []string{"build"}
	args = append(args, a.BuildArgs...)
	args = append(args, a.BuildTarget)

	// FIXME: is it possible to apply b.skipTests?
	cmd := exec.CommandContext(ctx, "bazel", args...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return "", fmt.Errorf("running command: %w", err)
	}

	bazelBin, err := bazelBin(ctx, workspace, a)
	if err != nil {
		return "", fmt.Errorf("getting path of bazel-bin: %w", err)
	}

	tarPath := filepath.Join(bazelBin, buildTarPath(a.BuildTarget))
	return tarPath, nil
}

func (b *Builder) loadImage(ctx context.Context, out io.Writer, tarPath string, a *latest.BazelArtifact, tag string) (string, error) {
	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", fmt.Errorf("opening image tarball: %w", err)
	}
	defer imageTar.Close()

	bazelTag := buildImageTag(a.BuildTarget)
	imageID, err := b.localDocker.Load(ctx, out, imageTar, bazelTag)
	if err != nil {
		return "", fmt.Errorf("loading image into docker daemon: %w", err)
	}

	if err := b.localDocker.Tag(ctx, imageID, tag); err != nil {
		return "", fmt.Errorf("tagging the image: %w", err)
	}

	return imageID, nil
}

func bazelBin(ctx context.Context, workspace string, a *latest.BazelArtifact) (string, error) {
	args := []string{"info", "bazel-bin"}
	args = append(args, a.BuildArgs...)

	cmd := exec.CommandContext(ctx, "bazel", args...)
	cmd.Dir = workspace

	buf, err := util.RunCmdOut(cmd)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(buf)), nil
}

func trimTarget(buildTarget string) string {
	// TODO(r2d4): strip off leading //:, bad
	trimmedTarget := strings.TrimPrefix(buildTarget, "//")
	// Useful if root target "//:target"
	trimmedTarget = strings.TrimPrefix(trimmedTarget, ":")

	return trimmedTarget
}

func buildTarPath(buildTarget string) string {
	tarPath := trimTarget(buildTarget)
	tarPath = strings.Replace(tarPath, ":", string(os.PathSeparator), 1)

	return tarPath
}

func buildImageTag(buildTarget string) string {
	imageTag := trimTarget(buildTarget)
	imageTag = strings.TrimPrefix(imageTag, ":")

	// TODO(r2d4): strip off trailing .tar, even worse
	imageTag = strings.TrimSuffix(imageTag, ".tar")

	if strings.Contains(imageTag, ":") {
		return fmt.Sprintf("bazel/%s", imageTag)
	}

	return fmt.Sprintf("bazel:%s", imageTag)
}
