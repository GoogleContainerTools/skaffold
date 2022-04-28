package cloudrun

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deploy "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"

	k8syaml "sigs.k8s.io/yaml"
)

// Deployer deploys code to Google Cloud Run.
type Deployer struct {
	logger log.Logger

	DefaultProject string
	Region         string

	Cfg deploy.Config

	// additional client options for connecting to Cloud Run, used for tests
	clientOptions []option.ClientOption
}

// NewDeployer creates a new Deployer for Cloud Run from the Skaffold deploy config.
func NewDeployer(labeller *label.DefaultLabeller, crDeploy *latest.CloudRunDeploy) (*Deployer, error) {
	return &Deployer{
		DefaultProject: crDeploy.DefaultProjectID,
		Region:         crDeploy.Region,
		logger:         &log.NoopLogger{},
	}, nil
}

// Deploy creates a Cloud Run service using the provided manifest.
func (d *Deployer) Deploy(ctx context.Context, out io.Writer, artifacts []graph.Artifact, manifests manifest.ManifestList) error {
	for _, manifest := range manifests {
		if err := d.deployToCloudRun(ctx, out, manifest); err != nil {
			return err
		}
	}
	return nil
}

// Dependencies list the files that would trigger a redeploy
func (d *Deployer) Dependencies() ([]string, error) {
	return []string{}, nil
}

// Cleanup deletes the created Cloud Run services
func (d *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool, manifests manifest.ManifestList) error {
	return d.deleteRunService(ctx, out, dryRun, manifests)
}

// Render writes out the k8s configs, we may want to support this with service configs in the future
// but it's not being implemented now
func (d *Deployer) Render(context.Context, io.Writer, []graph.Artifact, bool, string) error {
	return nil
}

// GetDebugger Get the Debugger for Cloud Run. Not supported by this deployer.
func (d *Deployer) GetDebugger() debug.Debugger {
	return &debug.NoopDebugger{}
}

// GetLogger Get the logger for the Cloud Run deploy.
func (d *Deployer) GetLogger() log.Logger {
	return d.logger
}

// GetAccessor gets a no-op accessor for Cloud Run.
func (d *Deployer) GetAccessor() access.Accessor {
	return &access.NoopAccessor{}
}

// GetSyncer gets the file syncer for Cloud Run. Not supported by this deployer.
func (d *Deployer) GetSyncer() sync.Syncer {
	return &sync.NoopSyncer{}
}

// TrackBuildArtifacts is not supported by this deployer.
func (d *Deployer) TrackBuildArtifacts([]graph.Artifact) {

}

// RegisterLocalImages is not supported by this deployer.
func (d *Deployer) RegisterLocalImages([]graph.Artifact) {

}

// GetStatusMonitor gets the resource that will monitor deployment status.
func (d *Deployer) GetStatusMonitor() status.Monitor {
	return &status.NoopMonitor{}
}

func (d *Deployer) deployToCloudRun(ctx context.Context, out io.Writer, manifest []byte) error {
	crclient, err := run.NewService(ctx, append(gcp.ClientOptions(ctx), d.clientOptions...)...)
	if err != nil {
		return sErrors.NewError(fmt.Errorf("Unable to create Cloud Run Client"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_GET_CLOUD_RUN_CLIENT_ERR,
		})
	}
	service := &run.Service{}
	if err = k8syaml.Unmarshal(manifest, service); err != nil {
		return sErrors.NewError(fmt.Errorf("Unable to unmarshal Cloud Run Service config"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	if service.Metadata.Namespace == "" {
		service.Metadata.Namespace = d.DefaultProject
	}

	// we need to strip "skaffold.dev" from the run-id label because gcp labels don't support domains
	runID, foundID := service.Metadata.Labels["skaffold.dev/run-id"]
	if foundID {
		delete(service.Metadata.Labels, "skaffold.dev/run-id")
		service.Metadata.Labels["run-id"] = runID
	}

	serviceJSON, err := service.MarshalJSON()
	output.Blue.Fprintf(out, "Deploying Cloud Run service:\n %v", string(serviceJSON))
	parent := fmt.Sprintf("projects/%s/locations/%s", service.Metadata.Namespace, d.Region)

	sName := fmt.Sprintf("%s/services/%s", parent, service.Metadata.Name)
	getCall := crclient.Projects.Locations.Services.Get(sName)
	_, err = getCall.Do()

	if err != nil {
		gErr, ok := err.(*googleapi.Error)
		if !ok || gErr.Code != http.StatusNotFound {
			return sErrors.NewError(fmt.Errorf("Error checking Cloud Run State"), &proto.ActionableErr{
				Message: err.Error(),
				ErrCode: proto.StatusCode_DEPLOY_CANCELLED,
			})
		}
		// This is a new service, we need to create it
		createCall := crclient.Projects.Locations.Services.Create(parent, service)
		_, err = createCall.Do()
	} else {
		replaceCall := crclient.Projects.Locations.Services.ReplaceService(sName, service)
		_, err = replaceCall.Do()
	}
	if err != nil {
		return sErrors.NewError(fmt.Errorf("Error deploying Cloud Run Service"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR,
		})
	}
	// register status monitor
	return nil
}

func (d *Deployer) deleteRunService(ctx context.Context, out io.Writer, dryRun bool, manifests manifest.ManifestList) error {
	if len(manifests) != 1 {
		return sErrors.NewError(fmt.Errorf("Unexpected manifest for Cloud Run"),
			&proto.ActionableErr{
				Message: "Cloud Run expected a single Service manifest.",
				ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
			})
	}
	service := &run.Service{}
	if err := k8syaml.Unmarshal(manifests[0], service); err != nil {
		return sErrors.NewError(fmt.Errorf("Unable to unmarshal Cloud Run Service config"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}

	var projectID string
	if service.Metadata.Namespace != "" {
		projectID = service.Metadata.Namespace
	} else {
		projectID = d.DefaultProject
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, d.Region)
	sName := fmt.Sprintf("%s/services/%s", parent, service.Metadata.Name)
	if dryRun {
		output.Yellow.Fprintln(out, sName)
		return nil
	}
	crclient, err := run.NewService(ctx, append(gcp.ClientOptions(ctx), d.clientOptions...)...)
	if err != nil {
		return sErrors.NewError(fmt.Errorf("Unable to create Cloud Run Client"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_GET_CLOUD_RUN_CLIENT_ERR,
		})
	}
	delCall := crclient.Projects.Locations.Services.Delete(sName)
	_, err = delCall.Do()
	if err != nil {
		return sErrors.NewError(fmt.Errorf("Unable to delete Cloud Run Service"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_DELETE_SERVICE_ERR,
		})
	}
	return nil
}
