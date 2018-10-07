package acr

import (
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
	"path/filepath"
)

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

	dockerFilePath := filepath.Join(artifact.Workspace, artifact.DockerArtifact.DockerfilePath)
	buildRequest := cr.DockerBuildRequest{
		ImageNames:     &[]string{imageTag},
		IsPushEnabled:  &[]bool{true}[0], //who invented bool pointers
		SourceLocation: &artifact.Workspace,
		Platform: &cr.PlatformProperties{
			Variant:      cr.V8,
			Os:           cr.Linux,
			Architecture: cr.Amd64,
		},
		DockerFilePath: &dockerFilePath,
		Type:           cr.TypeDockerBuildRequest,
	}
	future, err := client.ScheduleRun(ctx, b.ResourceGroup, b.ContainerRegistry, buildRequest)
	if err != nil {
		return "", errors.Wrap(err, "schedule build request")
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return "", errors.Wrap(err, "wait for build completion")
	}

	return imageTag, nil
}
