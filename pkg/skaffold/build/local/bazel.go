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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
)

func (b *Builder) buildBazel(ctx context.Context, out io.Writer, workspace string, a *latest.Artifact) (string, error) {
	args := []string{"build"}
	args = append(args, a.BazelArtifact.BuildArgs...)
	args = append(args, a.BazelArtifact.BuildTarget)

	cmd := exec.CommandContext(ctx, "bazel", args...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	bazelBin, err := bazelBin(ctx, workspace, a.BazelArtifact)
	if err != nil {
		return "", errors.Wrap(err, "getting path of bazel-bin")
	}

	tarPath := filepath.Join(bazelBin, buildTarPath(a.BazelArtifact.BuildTarget))

	if b.pushImages {
		uniqueTag := a.ImageName + ":" + util.RandomID()
		return pushImage(tarPath, uniqueTag)
	}

	ref := buildImageTag(a.BazelArtifact.BuildTarget)
	return b.loadImage(ctx, out, tarPath, ref)
}

func pushImage(tarPath, tag string) (string, error) {
	t, err := name.NewTag(tag, name.WeakValidation)
	if err != nil {
		return "", errors.Wrapf(err, "parsing tag %q", tag)
	}

	auth, err := authn.DefaultKeychain.Resolve(t.Registry)
	if err != nil {
		return "", errors.Wrapf(err, "getting creds for %q", t)
	}

	i, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return "", errors.Wrapf(err, "reading image %q", tarPath)
	}

	if err := remote.Write(t, i, auth, http.DefaultTransport); err != nil {
		return "", errors.Wrapf(err, "writing image %q", t)
	}

	return docker.RemoteDigest(tag)
}

func (b *Builder) loadImage(ctx context.Context, out io.Writer, tarPath string, ref string) (string, error) {
	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", errors.Wrap(err, "opening image tarball")
	}
	defer imageTar.Close()

	imageID, err := b.localDocker.Load(ctx, out, imageTar, ref)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
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
	//TODO(r2d4): strip off leading //:, bad
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

	//TODO(r2d4): strip off trailing .tar, even worse
	imageTag = strings.TrimSuffix(imageTag, ".tar")

	if strings.Contains(imageTag, ":") {
		return fmt.Sprintf("bazel/%s", imageTag)
	}

	return fmt.Sprintf("bazel:%s", imageTag)
}
