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

package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LocalBuilder uses the host docker daemon to build and tag the image
type LocalBuilder struct {
	*v1alpha2.BuildConfig

	api          docker.DockerAPIClient
	localCluster bool
	kubeContext  string
}

// NewLocalBuilder returns an new instance of a LocalBuilder
func NewLocalBuilder(cfg *v1alpha2.BuildConfig, kubeContext string) (*LocalBuilder, error) {
	api, err := docker.NewDockerAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "getting docker client")
	}

	l := &LocalBuilder{
		BuildConfig: cfg,

		kubeContext:  kubeContext,
		api:          api,
		localCluster: kubeContext == constants.DefaultMinikubeContext || kubeContext == constants.DefaultDockerForDesktopContext,
	}

	if cfg.LocalBuild.SkipPush == nil {
		logrus.Debugf("skipPush value not present. defaulting to cluster default %t (minikube=true, d4d=true, gke=false)", l.localCluster)
		cfg.LocalBuild.SkipPush = &l.localCluster
	}

	return l, nil
}

func (l *LocalBuilder) runBuildForArtifact(ctx context.Context, out io.Writer, artifact *v1alpha2.Artifact) (string, error) {
	if artifact.DockerArtifact != nil {
		return l.buildDocker(ctx, out, artifact)
	}
	if artifact.BazelArtifact != nil {
		return l.buildBazel(ctx, out, artifact)
	}

	return "", fmt.Errorf("undefined artifact type: %+v", artifact.ArtifactType)
}

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (l *LocalBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact) ([]Build, error) {
	if l.localCluster {
		if _, err := fmt.Fprintf(out, "Found [%s] context, using local docker daemon.\n", l.kubeContext); err != nil {
			return nil, errors.Wrap(err, "writing status")
		}
	}
	defer l.api.Close()

	var builds []Build

	for _, artifact := range artifacts {
		initialTag, err := l.runBuildForArtifact(ctx, out, artifact)
		if err != nil {
			return nil, errors.Wrap(err, "running build for artifact")
		}

		digest, err := docker.Digest(ctx, l.api, initialTag)
		if err != nil {
			return nil, errors.Wrapf(err, "build and tag: %s", initialTag)
		}
		if digest == "" {
			return nil, fmt.Errorf("digest not found")
		}
		tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, &tag.TagOptions{
			ImageName: artifact.ImageName,
			Digest:    digest,
		})
		if err != nil {
			return nil, errors.Wrap(err, "generating tag")
		}
		if err := l.api.ImageTag(ctx, initialTag, tag); err != nil {
			return nil, errors.Wrap(err, "tagging image")
		}
		if _, err := io.WriteString(out, fmt.Sprintf("Successfully tagged %s\n", tag)); err != nil {
			return nil, errors.Wrap(err, "writing tag status")
		}
		if !*l.LocalBuild.SkipPush {
			if err := docker.RunPush(ctx, l.api, tag, out); err != nil {
				return nil, errors.Wrap(err, "running push")
			}
		}

		builds = append(builds, Build{
			ImageName: artifact.ImageName,
			Tag:       tag,
			Artifact:  artifact,
		})
	}

	return builds, nil
}

func (l *LocalBuilder) buildDocker(ctx context.Context, out io.Writer, a *v1alpha2.Artifact) (string, error) {
	initialTag := util.RandomID()
	// Add a sanity check to check if the dockerfile exists before running the build
	if _, err := os.Stat(filepath.Join(a.Workspace, a.DockerArtifact.DockerfilePath)); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("Could not find dockerfile: %s", a.DockerArtifact.DockerfilePath)
		}
		return "", errors.Wrap(err, "stat dockerfile")
	}
	err := docker.RunBuild(ctx, l.api, &docker.BuildOptions{
		ImageName:   initialTag,
		Dockerfile:  a.DockerArtifact.DockerfilePath,
		ContextDir:  a.Workspace,
		ProgressBuf: out,
		BuildBuf:    out,
		BuildArgs:   a.DockerArtifact.BuildArgs,
	})
	if err != nil {
		return "", errors.Wrap(err, "running build")
	}
	return fmt.Sprintf("%s:latest", initialTag), nil
}
