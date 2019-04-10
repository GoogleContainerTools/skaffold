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
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Builder) localInit(runCtx *runcontext.RunContext) error {
	var l *latest.LocalBuild
	if err := util.CloneThroughJSON(b.runCtx.Cfg.Build.ExecutionEnvironment.Properties, &l); err != nil {
		return errors.Wrap(err, "converting execution env to localBuild struct")
	}
	if l == nil {
		l = &latest.LocalBuild{}
	}
	b.cfgLocal = local{}
	b.cfgLocal.LocalBuild = l

	b.KubeContext = runCtx.KubeContext
	localDocker, err := docker.NewAPIClient(b.runCtx.Opts.Prune())
	if err != nil {
		return errors.Wrap(err, "getting docker client")
	}
	b.cfgLocal.daemon = localDocker
	localCluster, err := configutil.GetLocalCluster()
	if err != nil {
		return errors.Wrap(err, "getting isLocalCluster")
	}
	b.cfgLocal.isLocalCluster = localCluster
	var pushImages bool
	if b.cfgLocal.LocalBuild.Push == nil {
		pushImages = !localCluster
		logrus.Debugf("push value not present, defaulting to %t because isLocalCluster is %t", pushImages, localCluster)
	} else {
		pushImages = *b.cfgLocal.LocalBuild.Push
	}
	b.cfgLocal.isPush = pushImages
	return nil
}

func (b *Builder) localBuild(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	for _, a := range artifacts {
		if err := setArtifact(a); err != nil {
			return nil, err
		}
	}
	return b.buildArtifacts(ctx, out, tags, artifacts)
}

func (b *Builder) prune(ctx context.Context, out io.Writer) error {
	return docker.Prune(ctx, out, b.builtImages, b.cfgLocal.daemon)
}

func (b *Builder) buildArtifacts(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if b.cfgLocal.isLocalCluster {
		color.Default.Fprintf(out, "Found [%s] context, using localBuild docker daemon.\n", b.KubeContext)
	}
	return build.InSequence(ctx, out, tags, artifacts, b.runBuild)
}

func (b *Builder) runBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	digestOrImageID, err := b.BuildArtifact(ctx, out, artifact, tag)
	if err != nil {
		return "", errors.Wrap(err, "build artifact")
	}
	if b.cfgLocal.isPush {
		imageID, err := b.getImageIDForTag(ctx, tag)
		if err != nil {
			logrus.Warnf("unable to inspect image: built images may not be cleaned up correctly by skaffold")
		}
		b.builtImages = append(b.builtImages, imageID)
		digest := digestOrImageID
		return tag + "@" + digest, nil
	}

	// k8s doesn't recognize the imageID or any combination of the image name
	// suffixed with the imageID, as a valid image name.
	// So, the solution we chose is to create a tag, just for Skaffold, from
	// the imageID, and use that in the manifests.
	imageID := digestOrImageID
	b.builtImages = append(b.builtImages, imageID)
	uniqueTag := artifact.ImageName + ":" + strings.TrimPrefix(imageID, "sha256:")
	if err := b.cfgLocal.daemon.Tag(ctx, imageID, uniqueTag); err != nil {
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

	fmt.Printf("%+v", b.cfgLocal)
	if b.cfgLocal.LocalBuild.UseDockerCLI || b.cfgLocal.LocalBuild.UseBuildkit {
		imageID, err = b.dockerCLIBuild(ctx, out, a.Workspace, a.ArtifactType.DockerArtifact, tag)
	} else {
		imageID, err = b.cfgLocal.daemon.Build(ctx, out, a.Workspace, a.ArtifactType.DockerArtifact, tag)
	}

	if err != nil {
		return "", err
	}

	if b.cfgLocal.isPush {
		return b.cfgLocal.daemon.Push(ctx, out, tag)
	}

	return imageID, nil
}

func (b *Builder) dockerCLIBuild(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, tag string) (string, error) {
	dockerfilePath, err := docker.NormalizeDockerfilePath(workspace, a.DockerfilePath)
	if err != nil {
		return "", errors.Wrap(err, "normalizing dockerfile path")
	}

	args := []string{"build", workspace, "--file", dockerfilePath, "-t", tag}
	args = append(args, docker.GetBuildArgs(a)...)
	if b.runCtx.Opts.Prune() {
		args = append(args, "--force-rm")
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	if b.cfgLocal.LocalBuild.UseBuildkit {
		cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := util.RunCmd(cmd); err != nil {
		return "", errors.Wrap(err, "running build")
	}

	return b.cfgLocal.daemon.ImageID(ctx, tag)
}

func (b *Builder) pullCacheFromImages(ctx context.Context, out io.Writer, a *latest.DockerArtifact) error {
	if len(a.CacheFrom) == 0 {
		return nil
	}

	for _, image := range a.CacheFrom {
		imageID, err := b.cfgLocal.daemon.ImageID(ctx, image)
		if err != nil {
			return errors.Wrapf(err, "getting imageID for %s", image)
		}
		if imageID != "" {
			// already pulled
			continue
		}

		if err := b.cfgLocal.daemon.Pull(ctx, out, image); err != nil {
			warnings.Printf("Cache-From image couldn't be pulled: %s\n", image)
		}
	}

	return nil
}

func (b *Builder) getImageIDForTag(ctx context.Context, tag string) (string, error) {
	insp, _, err := b.cfgLocal.daemon.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		return "", errors.Wrap(err, "inspecting image")
	}
	return insp.ID, nil
}
