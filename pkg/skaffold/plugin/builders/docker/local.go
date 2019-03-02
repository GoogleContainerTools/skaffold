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

package docker

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/pkg/errors"
)

func (b *Builder) local(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var l *latest.LocalBuild
	if err := util.CloneThroughJSON(b.env.Properties, &l); err != nil {
		return nil, errors.Wrap(err, "converting execution env to localBuild struct")
	}
	if l == nil {
		l = &latest.LocalBuild{}
	}
	b.LocalBuild = l
	kubeContext, err := kubectx.CurrentContext()
	if err != nil {
		return nil, errors.Wrap(err, "getting current cluster context")
	}
	b.KubeContext = kubeContext
	localDocker, err := docker.NewAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "getting docker client")
	}
	b.LocalDocker = localDocker
	for _, a := range artifacts {
		if err := setArtifact(a); err != nil {
			return nil, err
		}
	}
	return b.buildArtifacts(ctx, out, tags, artifacts)
}

func (b *Builder) buildArtifacts(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if b.LocalCluster {
		color.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.KubeContext)
	}
	return build.InSequence(ctx, out, tags, artifacts, b.runBuild)
}

func (b *Builder) runBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	digestOrImageID, err := b.BuildArtifact(ctx, out, artifact, tag)
	if err != nil {
		return "", errors.Wrap(err, "build artifact")
	}
	if b.PushImages {
		digest := digestOrImageID
		return tag + "@" + digest, nil
	}

	// k8s doesn't recognize the imageID or any combination of the image name
	// suffixed with the imageID, as a valid image name.
	// So, the solution we chose is to create a tag, just for Skaffold, from
	// the imageID, and use that in the manifests.
	imageID := digestOrImageID
	uniqueTag := artifact.ImageName + ":" + strings.TrimPrefix(imageID, "sha256:")
	if err := b.LocalDocker.Tag(ctx, imageID, uniqueTag); err != nil {
		return "", err
	}

	return uniqueTag, nil
}

// BuildArtifact builds the docker artifact
func (b *Builder) BuildArtifact(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	if err := b.pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact); err != nil {
		return "", errors.Wrap(err, "pulling cache-from images")
	}

	var (
		imageID string
		err     error
	)

	if b.LocalBuild.UseDockerCLI || b.LocalBuild.UseBuildkit {
		imageID, err = b.dockerCLIBuild(ctx, out, a.Workspace, a.ArtifactType.DockerArtifact, tag)
	} else {
		imageID, err = b.LocalDocker.Build(ctx, out, a.Workspace, a.ArtifactType.DockerArtifact, tag)
	}

	if b.PushImages {
		return b.LocalDocker.Push(ctx, out, tag)
	}

	return imageID, err
}

func (b *Builder) dockerCLIBuild(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, tag string) (string, error) {
	dockerfilePath, err := docker.NormalizeDockerfilePath(workspace, a.DockerfilePath)
	if err != nil {
		return "", errors.Wrap(err, "normalizing dockerfile path")
	}

	args := []string{"build", workspace, "--file", dockerfilePath, "-t", tag}
	args = append(args, docker.GetBuildArgs(a)...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	if b.LocalBuild.UseBuildkit {
		cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := util.RunCmd(cmd); err != nil {
		return "", errors.Wrap(err, "running build")
	}

	return b.LocalDocker.ImageID(ctx, tag)
}

func (b *Builder) pullCacheFromImages(ctx context.Context, out io.Writer, a *latest.DockerArtifact) error {
	if len(a.CacheFrom) == 0 {
		return nil
	}

	for _, image := range a.CacheFrom {
		imageID, err := b.LocalDocker.ImageID(ctx, image)
		if err != nil {
			return errors.Wrapf(err, "getting imageID for %s", image)
		}
		if imageID != "" {
			// already pulled
			continue
		}

		if err := b.LocalDocker.Pull(ctx, out, image); err != nil {
			warnings.Printf("Cache-From image couldn't be pulled: %s\n", image)
		}
	}

	return nil
}
