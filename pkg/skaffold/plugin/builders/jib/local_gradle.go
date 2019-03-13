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

package jib

import (
	"context"
	"io"
	"os"
	"strings"

	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// local sets any necessary defaults and then builds artifacts with Jib locally
func (b *GradleBuilder) local(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	localCluster, err := configutil.GetLocalCluster()
	if err != nil {
		return nil, errors.Wrap(err, "getting localCluster")
	}
	b.LocalCluster = localCluster
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
	var l *latest.LocalBuild
	if err := util.CloneThroughJSON(b.env.Properties, &l); err != nil {
		return nil, errors.Wrap(err, "converting execution env to localBuild struct")
	}
	if l == nil {
		l = &latest.LocalBuild{}
	}
	b.LocalBuild = l
	var pushImages bool
	if b.LocalBuild.Push == nil {
		pushImages = !localCluster
		logrus.Debugf("push value not present, defaulting to %t because localCluster is %t", pushImages, localCluster)
	} else {
		pushImages = *b.LocalBuild.Push
	}
	b.PushImages = pushImages
	for _, a := range artifacts {
		if err := setMavenArtifact(a); err != nil {
			return nil, errors.Wrapf(err, "setting artifact %s", a.ImageName)
		}
	}
	return b.buildArtifacts(ctx, out, tags, artifacts)
}

func (b *GradleBuilder) buildArtifacts(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if b.LocalCluster {
		color.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.KubeContext)
	}
	return build.InSequence(ctx, out, tags, artifacts, b.runBuild)
}

func (b *GradleBuilder) runBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
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

// BuildArtifact builds the Jib artifact
func (b *GradleBuilder) BuildArtifact(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	return b.buildJibGradle(ctx, out, artifact.Workspace, artifact.JibGradleArtifact, tag)
}

func (b *GradleBuilder) buildJibGradle(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibGradleArtifact, tag string) (string, error) {
	if b.PushImages {
		return b.buildJibGradleToRegistry(ctx, out, workspace, artifact, tag)
	}
	return b.buildJibGradleToDocker(ctx, out, workspace, artifact, tag)
}

func (b *GradleBuilder) buildJibGradleToDocker(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibGradleArtifact, tag string) (string, error) {
	args := jib.GenerateGradleArgs("jibDockerBuild", tag, artifact, b.opts.SkipTests)
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return b.LocalDocker.ImageID(ctx, tag)
}

func (b *GradleBuilder) buildJibGradleToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibGradleArtifact, tag string) (string, error) {
	args := jib.GenerateGradleArgs("jib", tag, artifact, b.opts.SkipTests)
	if err := b.runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return docker.RemoteDigest(tag)
}

func (b *GradleBuilder) runGradleCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	cmd := jib.GradleCommand.CreateCommand(ctx, workspace, args)
	cmd.Env = append(os.Environ(), b.LocalDocker.ExtraEnv()...)
	cmd.Stdout = out
	cmd.Stderr = out

	logrus.Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(cmd); err != nil {
		return errors.Wrap(err, "gradle build failed")
	}

	return nil
}
