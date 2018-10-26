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

package icr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/IBM-Cloud/bluemix-go/api/container/registryv1"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Build builds the image
func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	return build.InParallel(ctx, out, tagger, artifacts, b.buildArtifact)
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {

	var (
		registryClient     *IBMRegistrySession
		imageBuildRequest  registryv1.ImageBuildRequest
		imageTag           string
		buildArgBytes      []byte
		err                error
		buildCtx, pr       *io.PipeReader
		buildCtxWriter, pw *io.PipeWriter
		body               io.Reader
		progressOutput     progress.Output
		imageName          = artifact.ImageName
		workspace          = artifact.Workspace
		dockerArtifact     = artifact.DockerArtifact
	)

	if !reference.ReferenceRegexp.MatchString(imageName) {
		return "", errors.Errorf("Image Name is not correct format!")
	}

	registryClient, imageName, err = b.NewRegistryClient(imageName)
	if err != nil {
		return "", errors.Wrap(err, "Unable to Connect to IBM Cloud")
	}

	if !reference.ReferenceRegexp.MatchString(imageName) {
		return "", errors.Errorf("Image Name is not correct format!")
	}
	logrus.Debugf("Running IBM Container Registry build: context: %s, dockerfile: %s", workspace, dockerArtifact)

	imageTag, err = tagger.GenerateFullyQualifiedImageName(workspace, &tag.Options{
		Digest:    util.RandomID(),
		ImageName: imageName,
	})
	if err != nil {
		return "", errors.Wrap(err, "error generating image name")
	}
	buildCtx, buildCtxWriter = io.Pipe()
	go func() {
		if err := docker.CreateDockerTarContext(ctx, buildCtxWriter, workspace, dockerArtifact); err != nil {
			buildCtxWriter.CloseWithError(errors.Wrap(err, "creating docker context"))
			return
		}
		buildCtxWriter.Close()
	}()

	if dockerArtifact.BuildArgs != nil && len(dockerArtifact.BuildArgs) > 0 {
		buildArgBytes, err = json.Marshal(dockerArtifact.BuildArgs)
		if err != nil {
			return "", errors.Wrap(err, "Unable to marshal build args as json")
		}
	}

	imageBuildRequest = registryv1.ImageBuildRequest{
		T:          imageTag,
		Dockerfile: dockerArtifact.DockerfilePath,
		Buildargs:  fmt.Sprintf("%s", buildArgBytes),
	}

	progressOutput = streamformatter.NewProgressOutput(out)
	body = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to IBM Container Registry Build deamon")

	pr, pw = io.Pipe()
	go func() {
		if err := registryClient.Builds.ImageBuild(imageBuildRequest, body, registryClient.BuildTargetHeader, pw); err != nil {
			return
		}
		pw.Close()
	}()
	if err = docker.StreamDockerMessages(out, pr); err != nil {
		return "", errors.Wrap(err, "Unable to stream IBM Container Registry build messages")
	}

	return imageTag, nil
}
