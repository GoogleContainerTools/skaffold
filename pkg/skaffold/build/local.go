/*
Copyright 2018 Google LLC

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

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LocalBuilder uses the host docker daemon to build and tag the image
type LocalBuilder struct {
	*config.BuildConfig

	api          docker.DockerAPIClient
	localCluster bool
	kubeContext  string
}

// NewLocalBuilder returns an new instance of a LocalBuilder
func NewLocalBuilder(cfg *config.BuildConfig, kubeContext string) (*LocalBuilder, error) {
	if cfg.LocalBuild == nil {
		return nil, fmt.Errorf("LocalBuild config field is needed to create a new LocalBuilder")
	}

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

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (l *LocalBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*config.Artifact) (*BuildResult, error) {
	if l.localCluster {
		if _, err := fmt.Fprintf(out, "Found [%s] context, using local docker daemon.\n", l.kubeContext); err != nil {
			return nil, errors.Wrap(err, "writing status")
		}
	}
	defer l.api.Close()
	res := &BuildResult{
		Builds: []Build{},
	}
	for _, artifact := range artifacts {
		if artifact.DockerfilePath == "" {
			artifact.DockerfilePath = constants.DefaultDockerfilePath
		}
		initialTag := util.RandomID()
		err := docker.RunBuild(ctx, l.api, &docker.BuildOptions{
			ImageName:   initialTag,
			Dockerfile:  artifact.DockerfilePath,
			ContextDir:  artifact.Workspace,
			ProgressBuf: out,
			BuildBuf:    out,
			BuildArgs:   artifact.BuildArgs,
		})
		if err != nil {
			return nil, errors.Wrap(err, "running build")
		}
		digest, err := docker.Digest(ctx, l.api, initialTag)
		if err != nil {
			return nil, errors.Wrap(err, "build and tag")
		}
		if digest == "" {
			return nil, fmt.Errorf("digest not found")
		}
		tag, err := tagger.GenerateFullyQualifiedImageName(".", &tag.TagOptions{
			ImageName: artifact.ImageName,
			Digest:    digest,
		})
		if err != nil {
			return nil, errors.Wrap(err, "generating tag")
		}
		if err := l.api.ImageTag(ctx, fmt.Sprintf("%s:latest", initialTag), tag); err != nil {
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
		res.Builds = append(res.Builds, Build{
			ImageName: artifact.ImageName,
			Tag:       tag,
			Artifact:  artifact,
		})
	}

	return res, nil
}
