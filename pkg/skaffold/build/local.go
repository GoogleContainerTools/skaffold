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
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
)

// LocalBuilder uses the host docker daemon to build and tag the image
type LocalBuilder struct {
	*config.BuildConfig

	newAPI func() (client.ImageAPIClient, io.Closer, error)
}

// NewLocalBuilder returns an new instance of a LocalBuilder
func NewLocalBuilder(cfg *config.BuildConfig) (*LocalBuilder, error) {
	return &LocalBuilder{
		BuildConfig: cfg,
		newAPI:      docker.NewImageAPIClient,
	}, nil
}

// Run runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (l *LocalBuilder) Run(tagger tag.Tagger) (*BuildResult, error) {
	api, c, err := l.newAPI()
	if err != nil {
		return nil, errors.Wrap(err, "getting image api client")
	}
	defer c.Close()
	res := &BuildResult{
		Builds: []Build{},
	}
	for _, artifact := range l.Artifacts {
		if artifact.DockerfilePath == "" {
			artifact.DockerfilePath = constants.DefaultDockerfilePath
		}
		initialTag := util.RandomID()
		err := docker.RunBuild(api, &docker.BuildOptions{
			ImageName:   initialTag,
			Dockerfile:  artifact.DockerfilePath,
			ContextDir:  artifact.Workspace,
			ProgressBuf: util.Writer(),
			BuildBuf:    util.Writer(),
		})
		if err != nil {
			return nil, errors.Wrap(err, "running build")
		}
		digest, err := docker.Digest(api, initialTag)
		if err != nil {
			return nil, errors.Wrap(err, "build and tag")
		}
		if digest == "" {
			return nil, fmt.Errorf("digest not found")
		}
		tag, err := tagger.GenerateFullyQualifiedImageName(&tag.TagOptions{
			ImageName: artifact.ImageName,
			Digest:    digest,
		})
		if err != nil {
			return nil, errors.Wrap(err, "generating tag")
		}
		if err := api.ImageTag(context.Background(), fmt.Sprintf("%s:latest", initialTag), tag); err != nil {
			return nil, errors.Wrap(err, "tagging image")
		}
		util.Outputf("Successfully tagged %s", tag)

		if l.LocalBuild.Push {
			if err := docker.RunPush(api, tag); err != nil {
				return nil, errors.Wrap(err, "running push")
			}
		}
		res.Builds = append(res.Builds, Build{
			ImageName: artifact.ImageName,
			Tag:       tag,
		})
	}

	return res, nil
}
