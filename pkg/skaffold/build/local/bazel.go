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
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/builders/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

func (b *Builder) buildBazel(ctx context.Context, out io.Writer, workspace string, a *latest.BazelArtifact, tag string) (string, error) {
	args := []string{"build"}
	args = append(args, a.BuildArgs...)
	args = append(args, a.BuildTarget)

	// FIXME: is it possible to apply b.skipTests?
	cmd := exec.CommandContext(ctx, "bazel", args...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	bazelBin, err := bazel.BazelBin(ctx, workspace, a)
	if err != nil {
		return "", errors.Wrap(err, "getting path of bazel-bin")
	}

	tarPath := filepath.Join(bazelBin, bazel.BuildTarPath(a.BuildTarget))

	if b.pushImages {
		return bazel.PushImage(tarPath, tag)
	}

	return b.loadImage(ctx, out, tarPath, a, tag)
}

func (b *Builder) loadImage(ctx context.Context, out io.Writer, tarPath string, a *latest.BazelArtifact, tag string) (string, error) {
	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", errors.Wrap(err, "opening image tarball")
	}
	defer imageTar.Close()

	bazelTag := bazel.BuildImageTag(a.BuildTarget)
	imageID, err := b.localDocker.Load(ctx, out, imageTar, bazelTag)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
	}

	if err := b.localDocker.Tag(ctx, imageID, tag); err != nil {
		return "", errors.Wrap(err, "tagging the image")
	}

	return imageID, nil
}
