package acr

import (
	"bufio"
	"context"
	cr "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2018-09-01/containerregistry"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"regexp"
	"time"
)

const BUILD_STATUS_HEADER = "x-ms-meta-Complete"

func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	return build.InParallel(ctx, out, tagger, artifacts, b.buildArtifact)
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
	client := cr.NewRegistriesClient(b.Credentials.SubscriptionId)
	authorizer, err := auth.NewClientCredentialsConfig(b.Credentials.ClientId, b.Credentials.ClientSecret, b.Credentials.TenantId).Authorizer()
	if err != nil {
		return "", errors.Wrap(err, "authorizing client")
	}
	client.Authorizer = authorizer

	result, err := client.GetBuildSourceUploadURL(ctx, b.ResourceGroup, b.ContainerRegistry)
	if err != nil {
		return "", errors.Wrap(err, "build source upload url")
	}
	blob := NewBlobStorage(*result.UploadURL)

	err = docker.CreateDockerTarGzContext(blob.Writer(), artifact.Workspace, artifact.DockerArtifact)
	if err != nil {
		return "", errors.Wrap(err, "create context tar.gz")
	}

	err = blob.UploadFileToBlob()
	if err != nil {
		return "", errors.Wrap(err, "upload file to blob")
	}

	imageTag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, &tag.Options{
		Digest:    util.RandomID(),
		ImageName: artifact.ImageName,
	})
	if err != nil {
		return "", errors.Wrap(err, "create fully qualified image name")
	}

	imageTag, err = getImageTagWithoutFQDN(imageTag)
	if err != nil {
		return "", errors.Wrap(err, "get azure image tag")
	}

	buildRequest := cr.DockerBuildRequest{
		ImageNames:     &[]string{imageTag},
		IsPushEnabled:  &[]bool{true}[0], //who invented bool pointers
		SourceLocation: result.RelativePath,
		Platform: &cr.PlatformProperties{
			Variant:      cr.V8,
			Os:           cr.Linux,
			Architecture: cr.Amd64,
		},
		DockerFilePath: &artifact.DockerArtifact.DockerfilePath,
		Type:           cr.TypeDockerBuildRequest,
	}
	future, err := client.ScheduleRun(ctx, b.ResourceGroup, b.ContainerRegistry, buildRequest)
	if err != nil {
		return "", errors.Wrap(err, "schedule build request")
	}

	run, err := future.Result(client)
	if err != nil {
		return "", errors.Wrap(err, "get run id")
	}
	runId := *run.RunID

	runsClient := cr.NewRunsClient(b.Credentials.SubscriptionId)
	runsClient.Authorizer = client.Authorizer
	logUrl, err := runsClient.GetLogSasURL(ctx, b.ResourceGroup, b.ContainerRegistry, runId)
	if err != nil {
		return "", errors.Wrap(err, "get log url")
	}

	err = pollBuildStatus(*logUrl.LogLink, out)
	if err != nil {
		return "", errors.Wrap(err, "polling build status")
	}

	return imageTag, nil
}

func pollBuildStatus(logUrl string, out io.Writer) error {
	offset := int32(0)
	for {
		resp, err := http.Get(logUrl)
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
			if line > offset {
				out.Write(scanner.Bytes())
				offset++
			}
			line++
		}
		resp.Body.Close()

		if offset > 0 {
			switch resp.Header.Get(BUILD_STATUS_HEADER) {
			case "": //run succeeded when there is no status header
				return nil
			case "internalerror":
			case "failed":
				return errors.New("run failed")
			case "timedout":
				return errors.New("run timed out")
			case "canceled":
				return errors.New("run was canceled")
			}
		}

		time.Sleep(2 * time.Second)
	}
}

// ACR needs the image tag in the following format
// <registryName>/<repository>:<tag>
func getImageTagWithoutFQDN(imageTag string) (string, error) {
	r, err := regexp.Compile("(.*)\\..*\\..*(/.*)")
	if err != nil {
		return "", errors.Wrap(err, "create regexp")
	}

	matches := r.FindStringSubmatch(imageTag)
	if len(matches) < 3 {
		return "", errors.New("invalid image tag")
	}

	return matches[1] + matches[2], nil
}
