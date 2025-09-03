/*
Copyright 2022 The Skaffold Authors

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

package cloudrun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	logger "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

// Config contains config options needed for cloud run
type Config interface {
	PortForwardResources() []*latest.PortForwardResource
	PortForwardOptions() config.PortForwardOptions
	Mode() config.RunMode
	Tail() bool
}

// Deployer deploys code to Google Cloud Run. This implements the Deployer
// interface for Cloud Run.
type Deployer struct {
	configName string

	*latest.CloudRunDeploy

	logger              *LogAggregator
	accessor            *RunAccessor
	monitor             *Monitor
	labeller            *label.DefaultLabeller
	hookRunner          hooks.Runner
	statusCheckDeadline time.Duration
	// Whether or not to tolerate failures until the status check deadline is reached
	tolerateFailures   bool
	statusCheckEnabled *bool

	Project string
	Region  string

	// additional client options for connecting to Cloud Run, used for tests
	clientOptions []option.ClientOption
	useGcpOptions bool
}

// NewDeployer creates a new Deployer for Cloud Run from the Skaffold deploy config.
func NewDeployer(cfg Config, labeller *label.DefaultLabeller, crDeploy *latest.CloudRunDeploy, configName string, statusCheckDeadline time.Duration, tolerateFailures bool, statusCheckEnabled *bool) (*Deployer, error) {
	return &Deployer{
		configName:     configName,
		CloudRunDeploy: crDeploy,
		Project:        crDeploy.ProjectID,
		Region:         crDeploy.Region,
		// TODO: implement logger for Cloud Run.
		logger:              NewLoggerAggregator(cfg, labeller.GetRunID()),
		accessor:            NewAccessor(cfg, labeller.GetRunID()),
		labeller:            labeller,
		hookRunner:          hooks.NewCloudRunDeployRunner(crDeploy.LifecycleHooks, hooks.NewDeployEnvOpts(labeller.GetRunID(), "", []string{})),
		useGcpOptions:       true,
		statusCheckDeadline: statusCheckDeadline,
		tolerateFailures:    tolerateFailures,
		statusCheckEnabled:  statusCheckEnabled,
	}, nil
}

// Deploy creates a Cloud Run service using the provided manifest.
func (d *Deployer) Deploy(ctx context.Context, out io.Writer, artifacts []graph.Artifact, manifestsByConfig manifest.ManifestListByConfig) error {
	manifests := manifestsByConfig.GetForConfig(d.ConfigName())

	for _, manifest := range manifests {
		if err := d.deployToCloudRun(ctx, out, manifest); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) ConfigName() string {
	return d.configName
}

// Dependencies list the files that would trigger a redeploy
func (d *Deployer) Dependencies() ([]string, error) {
	return []string{}, nil
}

// Cleanup deletes the created Cloud Run services
func (d *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool, byConfig manifest.ManifestListByConfig) error {
	return d.cleanupRun(ctx, out, dryRun, byConfig.GetForConfig(d.configName))
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
	return d.accessor
}

// GetSyncer gets the file syncer for Cloud Run. Not supported by this deployer.
func (d *Deployer) GetSyncer() sync.Syncer {
	return &sync.NoopSyncer{}
}

// TrackBuildArtifacts is not supported by this deployer.
func (d *Deployer) TrackBuildArtifacts(_, _ []graph.Artifact) {

}

// RegisterLocalImages is not supported by this deployer.
func (d *Deployer) RegisterLocalImages([]graph.Artifact) {

}

// GetStatusMonitor gets the resource that will monitor deployment status.
func (d *Deployer) GetStatusMonitor() status.Monitor {
	statusCheckEnabled := d.statusCheckEnabled
	// assume disabled only if explicitly set to false. Status checking is turned
	// on by default
	if statusCheckEnabled != nil && !*statusCheckEnabled {
		return &status.NoopMonitor{}
	}
	return d.getMonitor()
}

func (d *Deployer) HasRunnableHooks() bool {
	return len(d.CloudRunDeploy.LifecycleHooks.PreHooks) > 0 || len(d.CloudRunDeploy.LifecycleHooks.PostHooks) > 0
}

func (d *Deployer) PreDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PreHooks")
	if err := d.hookRunner.RunPreHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func (d *Deployer) PostDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PostHooks")
	if err := d.hookRunner.RunPostHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func (d *Deployer) getMonitor() *Monitor {
	if d.monitor == nil {
		d.monitor = NewMonitor(d.labeller, d.clientOptions, d.statusCheckDeadline, d.tolerateFailures)
	}
	return d.monitor
}
func (d *Deployer) deployToCloudRun(ctx context.Context, out io.Writer, manifest []byte) error {
	cOptions := d.clientOptions
	if d.useGcpOptions {
		cOptions = append(cOptions, option.WithEndpoint(fmt.Sprintf("%s-run.googleapis.com", d.Region)))
		cOptions = append(gcp.ClientOptions(ctx), cOptions...)
	}
	crclient, err := run.NewService(ctx, cOptions...)
	if err != nil {
		return sErrors.NewError(fmt.Errorf("unable to create Cloud Run Client"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_GET_CLOUD_RUN_CLIENT_ERR,
		})
	}
	// figure out which type we have:
	resource := &unstructured.Unstructured{}
	if err = k8syaml.Unmarshal(manifest, resource); err != nil {
		return sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run Service config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	var resName *RunResourceName
	switch {
	case resource.GetAPIVersion() == "serving.knative.dev/v1" && resource.GetKind() == "Service":
		resName, err = d.deployService(crclient, manifest, out)
		// the accessor only supports services. Jobs don't run by themselves so port forwarding doesn't make sense.
		if resName != nil {
			d.accessor.AddResource(*resName)
		}
	case resource.GetAPIVersion() == "run.googleapis.com/v1" && resource.GetKind() == "Job":
		resName, err = d.deployJob(crclient, manifest, out)
	case resource.GetAPIVersion() == "run.googleapis.com/v1" && resource.GetKind() == "WorkerPool":
		resName, err = d.deployWorkerPool(crclient, manifest, out)
	default:
		err = sErrors.NewError(fmt.Errorf("unsupported Kind for Cloud Run Deployer: %s/%s", resource.GetAPIVersion(), resource.GetKind()),
			&proto.ActionableErr{
				Message: "Kind is not supported",
				ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
			})
	}

	if err != nil {
		return err
	}

	d.getMonitor().Resources = append(d.getMonitor().Resources, *resName)
	return nil
}

func (d *Deployer) deployService(crclient *run.APIService, manifest []byte, out io.Writer) (*RunResourceName, error) {
	service := &run.Service{}
	if err := k8syaml.Unmarshal(manifest, service); err != nil {
		return nil, sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run Service config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	if d.Project != "" {
		service.Metadata.Namespace = d.Project
	} else if service.Metadata.Namespace == "" {
		return nil, sErrors.NewError(fmt.Errorf("unable to detect project for Cloud Run"), &proto.ActionableErr{
			Message: "No Google Cloud project found in Cloud Run Manifest or Skaffold Config",
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	// we need to strip "skaffold.dev" from the run-id label because gcp labels don't support domains
	runID, foundID := service.Metadata.Labels["skaffold.dev/run-id"]
	if foundID {
		delete(service.Metadata.Labels, "skaffold.dev/run-id")
		service.Metadata.Labels["run-id"] = runID
	}
	if service.Spec != nil && service.Spec.Template != nil && service.Spec.Template.Metadata != nil {
		runID, foundID = service.Spec.Template.Metadata.Labels["skaffold.dev/run-id"]
		if foundID {
			delete(service.Spec.Template.Metadata.Labels, "skaffold.dev/run-id")
			service.Spec.Template.Metadata.Labels["run-id"] = runID
		}
	}
	resName := RunResourceName{
		Project: service.Metadata.Namespace,
		Region:  d.Region,
		Service: service.Metadata.Name,
	}
	output.Default.Fprintln(out, "Deploying Cloud Run service:\n\t", service.Metadata.Name)
	parent := fmt.Sprintf("projects/%s/locations/%s", service.Metadata.Namespace, d.Region)

	sName := resName.String()
	d.logger.AddResource(resName)
	getCall := crclient.Projects.Locations.Services.Get(sName)
	_, err := getCall.Do()

	if err != nil {
		gErr, ok := err.(*googleapi.Error)
		if !ok || gErr.Code != http.StatusNotFound {
			return nil, sErrors.NewError(fmt.Errorf("error checking Cloud Run State: %w", err), &proto.ActionableErr{
				Message: err.Error(),
				ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
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
		return nil, sErrors.NewError(fmt.Errorf("error deploying Cloud Run Service: %s", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR,
		})
	}
	return &resName, nil
}

func (d *Deployer) forceSendValueOfMaxRetries(job *run.Job, manifest []byte) {
	maxRetriesPath := []string{"spec", "template", "spec", "template", "spec"}
	node := make(map[string]interface{})

	if err := k8syaml.Unmarshal(manifest, &node); err != nil {
		logger.Entry(context.TODO()).Debugf("Error unmarshaling job into map, skipping maxRetries ForceSendFields logic: %v", err)
		return
	}

	for _, field := range maxRetriesPath {
		value := node[field]
		child, ok := value.(map[string]interface{})
		if !ok {
			logger.Entry(context.TODO()).Debugf("Job maxRetries parent fields not found")
			return
		}
		node = child
	}

	if _, exists := node["maxRetries"]; !exists {
		logger.Entry(context.TODO()).Debugf("Job maxRetries property not found")
		return
	}

	if job.Spec == nil || job.Spec.Template == nil || job.Spec.Template.Spec == nil || job.Spec.Template.Spec.Template == nil || job.Spec.Template.Spec.Template.Spec == nil {
		logger.Entry(context.TODO()).Debugf("Job struct doesn't have the required values to force maxRetries sending")
		return
	}
	job.Spec.Template.Spec.Template.Spec.ForceSendFields = append(job.Spec.Template.Spec.Template.Spec.ForceSendFields, "MaxRetries")
}

func (d *Deployer) deployJob(crclient *run.APIService, manifest []byte, out io.Writer) (*RunResourceName, error) {
	job := &run.Job{}
	if err := k8syaml.Unmarshal(manifest, job); err != nil {
		return nil, sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run Job config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}

	d.forceSendValueOfMaxRetries(job, manifest)

	if d.Project != "" {
		job.Metadata.Namespace = d.Project
	} else if job.Metadata.Namespace == "" {
		return nil, sErrors.NewError(fmt.Errorf("unable to detect project for Cloud Run"), &proto.ActionableErr{
			Message: "No Google Cloud project found in Cloud Run Manifest or Skaffold Config",
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	// we need to strip "skaffold.dev" from the run-id label because gcp labels don't support domains
	runID, foundID := job.Metadata.Labels["skaffold.dev/run-id"]
	if foundID {
		delete(job.Metadata.Labels, "skaffold.dev/run-id")
		job.Metadata.Labels["run-id"] = runID
	}
	if job.Spec != nil && job.Spec.Template != nil && job.Spec.Template.Metadata != nil {
		runID, foundID = job.Spec.Template.Metadata.Labels["skaffold.dev/run-id"]
		if foundID {
			delete(job.Spec.Template.Metadata.Labels, "skaffold.dev/run-id")
			job.Spec.Template.Metadata.Labels["run-id"] = runID
		}
	}
	resName := RunResourceName{
		Project: job.Metadata.Namespace,
		Region:  d.Region,
		Job:     job.Metadata.Name,
	}
	output.Default.Fprintln(out, "Deploying Cloud Run service:\n\t", job.Metadata.Name)
	parent := fmt.Sprintf("namespaces/%s", job.Metadata.Namespace)

	sName := resName.String()
	getCall := crclient.Namespaces.Jobs.Get(sName)
	_, err := getCall.Do()

	if err != nil {
		gErr, ok := err.(*googleapi.Error)
		if !ok || gErr.Code != http.StatusNotFound {
			return nil, sErrors.NewError(fmt.Errorf("error checking Cloud Run State: %w", err), &proto.ActionableErr{
				Message: err.Error(),
				ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_GET_SERVICE_ERR,
			})
		}
		// This is a new service, we need to create it
		createCall := crclient.Namespaces.Jobs.Create(parent, job)
		_, err = createCall.Do()
	} else {
		replaceCall := crclient.Namespaces.Jobs.ReplaceJob(sName, job)
		_, err = replaceCall.Do()
	}
	if err != nil {
		return nil, sErrors.NewError(fmt.Errorf("error deploying Cloud Run Job: %s", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR,
		})
	}
	return &resName, nil
}

func (d *Deployer) deployWorkerPool(crclient *run.APIService, manifest []byte, out io.Writer) (*RunResourceName, error) {
	workerpool := &run.WorkerPool{}
	if err := k8syaml.Unmarshal(manifest, workerpool); err != nil {
		return nil, sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run WorkerPool config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}

	if d.Project != "" {
		workerpool.Metadata.Namespace = d.Project
	} else if workerpool.Metadata.Namespace == "" {
		return nil, sErrors.NewError(fmt.Errorf("unable to detect project for Cloud Run"), &proto.ActionableErr{
			Message: "No Google Cloud project found in Cloud Run Manifest or Skaffold Config",
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	// we need to strip "skaffold.dev" from the run-id label because gcp labels don't support domains
	runID, foundID := workerpool.Metadata.Labels["skaffold.dev/run-id"]
	if foundID {
		delete(workerpool.Metadata.Labels, "skaffold.dev/run-id")
		workerpool.Metadata.Labels["run-id"] = runID
	}
	if workerpool.Spec != nil && workerpool.Spec.Template != nil && workerpool.Spec.Template.Metadata != nil {
		runID, foundID = workerpool.Spec.Template.Metadata.Labels["skaffold.dev/run-id"]
		if foundID {
			delete(workerpool.Spec.Template.Metadata.Labels, "skaffold.dev/run-id")
			workerpool.Spec.Template.Metadata.Labels["run-id"] = runID
		}
	}
	resName := RunResourceName{
		Project:    workerpool.Metadata.Namespace,
		Region:     d.Region,
		WorkerPool: workerpool.Metadata.Name,
	}
	output.Default.Fprintln(out, "Deploying Cloud Run WorkerPool:\n\t", workerpool.Metadata.Name)
	parent := fmt.Sprintf("namespaces/%s", workerpool.Metadata.Namespace)

	wpName := resName.String()
	getCall := crclient.Namespaces.Workerpools.Get(wpName)
	_, err := getCall.Do()

	if err != nil {
		gErr, ok := err.(*googleapi.Error)
		if !ok || gErr.Code != http.StatusNotFound {
			return nil, sErrors.NewError(fmt.Errorf("error checking Cloud Run State: %w", err), &proto.ActionableErr{
				Message: err.Error(),
				ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_GET_WORKER_POOL_ERR,
			})
		}
		// This is a new workerpool, we need to create it
		createCall := crclient.Namespaces.Workerpools.Create(parent, workerpool)
		_, err = createCall.Do()
	} else {
		replaceCall := crclient.Namespaces.Workerpools.ReplaceWorkerPool(wpName, workerpool)
		_, err = replaceCall.Do()
	}
	if err != nil {
		return nil, sErrors.NewError(fmt.Errorf("error deploying Cloud Run WorkerPool: %s", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_UPDATE_WORKER_POOL_ERR,
		})
	}
	return &resName, nil
}

func (d *Deployer) cleanupRun(ctx context.Context, out io.Writer, dryRun bool, manifests manifest.ManifestList) error {
	var errors []error
	cOptions := d.clientOptions
	if d.useGcpOptions {
		cOptions = append(cOptions, option.WithEndpoint(fmt.Sprintf("%s-run.googleapis.com", d.Region)))
		cOptions = append(gcp.ClientOptions(ctx), cOptions...)
	}
	crclient, err := run.NewService(ctx, cOptions...)
	if err != nil {
		return sErrors.NewError(fmt.Errorf("unable to create Cloud Run Client"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_GET_CLOUD_RUN_CLIENT_ERR,
		})
	}
	for _, manifest := range manifests {
		tpe, err := getTypeFromManifest(manifest)
		switch {
		case err != nil:
			errors = append(errors, err)
		case tpe == typeService:
			err := d.deleteRunService(crclient, out, dryRun, manifest)
			if err != nil {
				errors = append(errors, err)
			}
		case tpe == typeJob:
			err := d.deleteRunJob(crclient, out, dryRun, manifest)
			if err != nil {
				errors = append(errors, err)
			}
		case tpe == typeWorkerPool:
			err := d.deleteRunWorkerPool(crclient, out, dryRun, manifest)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) != 0 {
		// TODO: is there a good way to report all of the errors?
		return errors[0]
	}
	return nil
}

func (d *Deployer) deleteRunService(crclient *run.APIService, out io.Writer, dryRun bool, manifest []byte) error {
	service := &run.Service{}
	if err := k8syaml.Unmarshal(manifest, service); err != nil {
		return sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run Service config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}

	var projectID string
	switch {
	case d.Project != "":
		projectID = d.Project
	case service.Metadata.Namespace != "":
		projectID = service.Metadata.Namespace
	default:
		// no project specified, we don't know what to delete.
		return sErrors.NewError(fmt.Errorf("unable to determine Google Cloud Project"), &proto.ActionableErr{
			Message: "No Google Cloud Project found in Cloud Run manifest or Skaffold Manifest.",
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, d.Region)
	sName := fmt.Sprintf("%s/services/%s", parent, service.Metadata.Name)
	if dryRun {
		output.Yellow.Fprintln(out, sName)
		return nil
	}

	delCall := crclient.Projects.Locations.Services.Delete(sName)
	_, err := delCall.Do()
	if err != nil {
		return sErrors.NewError(fmt.Errorf("unable to delete Cloud Run Service"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_DELETE_SERVICE_ERR,
		})
	}
	return nil
}

func (d *Deployer) deleteRunJob(crclient *run.APIService, out io.Writer, dryRun bool, manifest []byte) error {
	job := &run.Job{}
	if err := k8syaml.Unmarshal(manifest, job); err != nil {
		return sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run Job config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	var projectID string
	switch {
	case d.Project != "":
		projectID = d.Project
	case job.Metadata.Namespace != "":
		projectID = job.Metadata.Namespace
	default:
		// no project specified, we don't know what to delete.
		return sErrors.NewError(fmt.Errorf("unable to determine Google Cloud Project"), &proto.ActionableErr{
			Message: "No Google Cloud Project found in Cloud Run manifest or Skaffold Manifest.",
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	parent := fmt.Sprintf("namespaces/%s", projectID)
	sName := fmt.Sprintf("%s/jobs/%s", parent, job.Metadata.Name)
	if dryRun {
		output.Yellow.Fprintln(out, sName)
		return nil
	}

	delCall := crclient.Namespaces.Jobs.Delete(sName)
	_, err := delCall.Do()
	if err != nil {
		return sErrors.NewError(fmt.Errorf("unable to delete Cloud Run Job"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_DELETE_SERVICE_ERR,
		})
	}
	return nil
}

func (d *Deployer) deleteRunWorkerPool(crclient *run.APIService, out io.Writer, dryRun bool, manifest []byte) error {
	workerpool := &run.WorkerPool{}
	if err := k8syaml.Unmarshal(manifest, workerpool); err != nil {
		return sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run WorkerPool config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}

	var projectID string
	switch {
	case d.Project != "":
		projectID = d.Project
	case workerpool.Metadata.Namespace != "":
		projectID = workerpool.Metadata.Namespace
	default:
		// no project specified, we don't know what to delete.
		return sErrors.NewError(fmt.Errorf("unable to determine Google Cloud Project"), &proto.ActionableErr{
			Message: "No Google Cloud Project found in Cloud Run manifest or Skaffold Manifest.",
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	parent := fmt.Sprintf("namespaces/%s", projectID)
	sName := fmt.Sprintf("%s/workerpools/%s", parent, workerpool.Metadata.Name)
	if dryRun {
		output.Yellow.Fprintln(out, sName)
		return nil
	}

	delCall := crclient.Namespaces.Workerpools.Delete(sName)
	_, err := delCall.Do()
	if err != nil {
		return sErrors.NewError(fmt.Errorf("unable to delete Cloud Run WorkerPool"), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLOUD_RUN_DELETE_WORKER_POOL_ERR,
		})
	}
	return nil
}

func getTypeFromManifest(manifest []byte) (string, error) {
	resource := &unstructured.Unstructured{}
	if err := k8syaml.Unmarshal(manifest, resource); err != nil {
		return "", sErrors.NewError(fmt.Errorf("unable to unmarshal Cloud Run Service config: %w", err), &proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
	}
	switch {
	case resource.GetAPIVersion() == "serving.knative.dev/v1" && resource.GetKind() == "Service":
		return typeService, nil
	case resource.GetAPIVersion() == "run.googleapis.com/v1" && resource.GetKind() == "Job":
		return typeJob, nil
	case resource.GetAPIVersion() == "run.googleapis.com/v1" && resource.GetKind() == "WorkerPool":
		return typeWorkerPool, nil
	default:
		err := sErrors.NewError(fmt.Errorf("unsupported Kind for Cloud Run Deployer: %s/%s", resource.GetAPIVersion(), resource.GetKind()),
			&proto.ActionableErr{
				Message: "Kind is not supported",
				ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
			})
		return "", err
	}
}
