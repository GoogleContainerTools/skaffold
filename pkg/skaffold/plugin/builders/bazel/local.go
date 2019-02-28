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
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
)

// local sets any necessary defaults and then builds artifacts with bazel locally
func (b *Builder) local(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var l *latest.LocalBuild
	if err := util.CloneThroughJSON(b.env.Properties, &l); err != nil {
		return nil, errors.Wrap(err, "converting execution env to localBuild struct")
	}
	if l == nil {
		l = &latest.LocalBuild{}
	}
	if l.Push != nil {
		b.pushImages = *l.Push
	}
	for _, a := range artifacts {
		if err := setArtifact(a); err != nil {
			return nil, errors.Wrapf(err, "setting artifact %s", a.ImageName)
		}
	}
	return b.buildArtifacts(ctx, out, tags, artifacts)
}

func (b *Builder) buildArtifacts(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if b.localCluster {
		color.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.kubeContext)
	}
	return build.InSequence(ctx, out, tags, artifacts, b.buildArtifact)
}

// buildArtifact builds the bazel artifact
func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	args := []string{"build"}
	a := artifact.ArtifactType.BazelArtifact
	workspace := artifact.Workspace
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

	bazelBin, err := BazelBin(ctx, workspace, a)
	if err != nil {
		return "", errors.Wrap(err, "getting path of bazel-bin")
	}

	tarPath := filepath.Join(bazelBin, BuildTarPath(a.BuildTarget))

	if b.pushImages {
		return PushImage(tarPath, tag)
	}

	return b.loadImage(ctx, out, tarPath, a, tag)
}

func PushImage(tarPath, tag string) (string, error) {
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

func (b *Builder) loadImage(ctx context.Context, out io.Writer, tarPath string, a *latest.BazelArtifact, tag string) (string, error) {
	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", errors.Wrap(err, "opening image tarball")
	}
	defer imageTar.Close()

	bazelTag := BuildImageTag(a.BuildTarget)
	imageID, err := b.localDocker.Load(ctx, out, imageTar, bazelTag)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
	}

	if err := b.localDocker.Tag(ctx, imageID, tag); err != nil {
		return "", errors.Wrap(err, "tagging the image")
	}

	return imageID, nil
}

func BazelBin(ctx context.Context, workspace string, a *latest.BazelArtifact) (string, error) {
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

func TrimTarget(buildTarget string) string {
	//TODO(r2d4): strip off leading //:, bad
	trimmedTarget := strings.TrimPrefix(buildTarget, "//")
	// Useful if root target "//:target"
	trimmedTarget = strings.TrimPrefix(trimmedTarget, ":")

	return trimmedTarget
}

func BuildTarPath(buildTarget string) string {
	tarPath := TrimTarget(buildTarget)
	tarPath = strings.Replace(tarPath, ":", string(os.PathSeparator), 1)

	return tarPath
}

func BuildImageTag(buildTarget string) string {
	imageTag := TrimTarget(buildTarget)
	imageTag = strings.TrimPrefix(imageTag, ":")

	//TODO(r2d4): strip off trailing .tar, even worse
	imageTag = strings.TrimSuffix(imageTag, ".tar")

	if strings.Contains(imageTag, ":") {
		return fmt.Sprintf("bazel/%s", imageTag)
	}

	return fmt.Sprintf("bazel:%s", imageTag)
}
