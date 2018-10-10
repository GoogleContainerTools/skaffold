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

package acr

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	cr "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2018-09-01/containerregistry"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

const BuildStatusHeader = "x-ms-meta-Complete"

func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	return build.InParallel(ctx, out, tagger, artifacts, b.buildArtifact)
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
	client := cr.NewRegistriesClient(b.Credentials.SubscriptionID)
	authorizer, err := auth.NewClientCredentialsConfig(b.Credentials.ClientID, b.Credentials.ClientSecret, b.Credentials.TenantID).Authorizer()
	if err != nil {
		return "", errors.Wrap(err, "authorizing client")
	}
	client.Authorizer = authorizer

	imageTag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, &tag.Options{
		Digest:    util.RandomID(),
		ImageName: artifact.ImageName,
	})
	if err != nil {
		return "", errors.Wrap(err, "create fully qualified image name")
	}
	registryName := getRegistryName(imageTag)

	result, err := client.GetBuildSourceUploadURL(ctx, b.ResourceGroup, registryName)
	if err != nil {
		return "", errors.Wrap(err, "build source upload url")
	}
	blob := NewBlobStorage(*result.UploadURL)

	err = docker.CreateDockerTarGzContext(ctx, blob.Buffer, artifact.Workspace, artifact.DockerArtifact)
	if err != nil {
		return "", errors.Wrap(err, "create context tar.gz")
	}

	err = blob.UploadFileToBlob()
	if err != nil {
		return "", errors.Wrap(err, "upload file to blob")
	}

	//acr needs the image tag formatted as <repository>:<tag>
	imageTag = getImageTagWithoutFQDN(imageTag)

	buildRequest := cr.DockerBuildRequest{
		ImageNames:     &[]string{imageTag},
		IsPushEnabled:  util.BoolPtr(true),
		SourceLocation: result.RelativePath,
		Platform: &cr.PlatformProperties{
			Variant:      cr.V8,
			Os:           cr.Linux,
			Architecture: cr.Amd64,
		},
		DockerFilePath: &artifact.DockerArtifact.DockerfilePath,
		Type:           cr.TypeDockerBuildRequest,
	}
	future, err := client.ScheduleRun(ctx, b.ResourceGroup, registryName, buildRequest)
	if err != nil {
		return "", errors.Wrap(err, "schedule build request")
	}

	run, err := future.Result(client)
	if err != nil {
		return "", errors.Wrap(err, "get run id")
	}
	runID := *run.RunID

	runsClient := cr.NewRunsClient(b.Credentials.SubscriptionID)
	runsClient.Authorizer = client.Authorizer
	logURL, err := runsClient.GetLogSasURL(ctx, b.ResourceGroup, registryName, runID)
	if err != nil {
		return "", errors.Wrap(err, "get log url")
	}

	err = streamBuildLogs(*logURL.LogLink, out)
	if err != nil {
		return "", errors.Wrap(err, "polling build status")
	}

	return imageTag, nil
}

func streamBuildLogs(logURL string, out io.Writer) error {
	offset := int32(0)
	for {
		resp, err := http.Get(logURL)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusNotFound {
			//if blob is not available yet, try again
			time.Sleep(2 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(resp.Body)
		line := int32(0)
		for scanner.Scan() {
			if line >= offset {
				out.Write(scanner.Bytes())
				out.Write([]byte("\n"))
				offset++
			}
			line++
		}
		resp.Body.Close()

		if offset > 0 {
			switch resp.Header.Get(BuildStatusHeader) {
			case "":
				continue
			case "internalerror":
			case "failed":
				return errors.New("run failed")
			case "timedout":
				return errors.New("run timed out")
			case "canceled":
				return errors.New("run was canceled")
			default:
				return nil
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func getImageTagWithoutFQDN(imageTag string) string {
	return imageTag[strings.Index(imageTag, "/")+1:]
}

//acr URL is <registryname>.azurecr.io
func getRegistryName(imageTag string) string {
	return imageTag[:strings.Index(imageTag, ".")]
}
