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
	"os"
	"os/exec"
	"strconv"

	"golang.org/x/sync/singleflight"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

var (
	retrieveAvailablePort = util.GetAvailablePort
	gcloudInstalled       = findGcloud
)

type resourceTracker struct {
	resources          map[RunResourceName]*forwardedResource
	configuredForwards []*latest.PortForwardResource
	forwardedPorts     *util.PortSet
}

type forwardedResource struct {
	name    RunResourceName
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	started bool
	port    int
}

type forwarder interface {
	// Start initiates the forwarder's operation. It should not return until any ports have been allocated.
	Start(ctx context.Context, out io.Writer) error
	Stop()
}

// RunAccessor is an access.Accessor for Cloud Run resources
// It uses `gcloud run proxy“to enable port forwarding for Cloud Run. This makes it easier to call IAM-protected Cloud Run services
// by going through localhost. In order to set up forwarding, the services must have their ingress setting set to "all", gcloud  must be
// installed and on the path, and the currently configured gcloud user has run.services.invoke permission on the services being proxied
type RunAccessor struct {
	resources  *resourceTracker
	forwarders []forwarder
	singleRun  singleflight.Group
	label      string
}

// NewAccessor creates a new RunAccessor to port forward Cloud Run services
func NewAccessor(cfg Config, label string) *RunAccessor {
	var forwarders []forwarder
	resources := &resourceTracker{forwardedPorts: &util.PortSet{}, configuredForwards: cfg.PortForwardResources()}
	options := cfg.PortForwardOptions()

	if options.ForwardServices(cfg.Mode()) {
		forwarders = append(forwarders, &runProxyForwarder{resources: resources})
	}
	return &RunAccessor{
		resources: resources, forwarders: forwarders, label: label, singleRun: singleflight.Group{}}
}

// AddResource tracks an additional resource to port forward
func (r *RunAccessor) AddResource(resource RunResourceName) {
	if r.resources.resources == nil {
		r.resources.resources = make(map[RunResourceName]*forwardedResource)
	}
	port := 0
	for _, forward := range r.resources.configuredForwards {
		if forward.Type == "service" && forward.Name == resource.Service {
			port = forward.LocalPort
		}
	}
	if _, present := r.resources.resources[resource]; !present {
		r.resources.resources[resource] = &forwardedResource{name: resource, started: false, port: port}
	} else {
		// signal that we need to start a new forward for this resource
		r.resources.resources[resource].started = false
	}
}

// Start begins port forwarding for the tracked Cloud Run services.
func (r *RunAccessor) Start(ctx context.Context, out io.Writer) error {
	if r == nil {
		return nil
	}
	_, err, _ := r.singleRun.Do(r.label, func() (interface{}, error) {
		return struct{}{}, r.start(ctx, out)
	})
	return err
}

func (r *RunAccessor) start(ctx context.Context, out io.Writer) error {
	for _, forwarder := range r.forwarders {
		if err := forwarder.Start(ctx, out); err != nil {
			return err
		}
	}
	return nil
}

// Stop terminates port forwarding for all tracked Cloud Run resources.
func (r *RunAccessor) Stop() {
	for _, forwarder := range r.forwarders {
		forwarder.Stop()
	}
}

func findGcloud() bool {
	cmd := exec.Command("gcloud", "--version")
	return cmd.Run() == nil
}

type runProxyForwarder struct {
	resources *resourceTracker
}

func (r *runProxyForwarder) Start(ctx context.Context, out io.Writer) error {
	if !gcloudInstalled() {
		output.Red.Fprintln(out, "gcloud not found on path. Unable to set up Cloud Run port forwarding")
		return sErrors.NewError(fmt.Errorf("gcloud not found"), &proto.ActionableErr{ErrCode: proto.StatusCode_PORT_FORWARD_RUN_GCLOUD_NOT_FOUND})
	}
	if r.resources.resources == nil {
		// no forwards configured
		return nil
	}
	for _, resource := range r.resources.resources {
		if resource.port == 0 {
			port := retrieveAvailablePort("localhost", 8080, r.resources.forwardedPorts)
			resource.port = port
		}
		if !resource.started {
			eventV2.TaskInProgress(constants.PortForward, "port forward URLs")
			// has not been started yet
			cctx, cancel := context.WithCancel(ctx)
			output.Yellow.Fprintf(out, "Forwarding service %s to local port %d\n", resource.name.String(), resource.port)
			cmd := exec.CommandContext(cctx, "gcloud", getGcloudProxyArgs(resource.name, resource.port)...)
			cmd.Stdout = out
			cmd.Stderr = out
			resource.cancel = cancel
			if err := cmd.Start(); err != nil {
				eventV2.TaskFailed(constants.PortForward, err)
				return sErrors.NewError(fmt.Errorf("unable to start port forward: %w", err), &proto.ActionableErr{ErrCode: proto.StatusCode_PORT_FORWARD_RUN_PROXY_START_ERROR})
			}
			go func() {
				err := cmd.Wait()
				if err != nil {
					eventV2.TaskFailed(constants.PortForward, err)
				} else {
					eventV2.TaskSucceeded(constants.PortForward)
				}
			}()
			eventV2.PortForwarded(int32(resource.port), schemautil.FromInt(443), "", "", resource.name.Project, "", "run-service", resource.name.Service, resource.name.String())
			resource.started = true
			resource.cmd = cmd
		}
	}
	return nil
}

func getGcloudProxyArgs(resource RunResourceName, port int) []string {
	return []string{"beta", "run", "services", "proxy", "--project", resource.Project, "--region", resource.Region, "--port", strconv.Itoa(port), resource.Service}
}

// Stop terminates port forwarding for all tracked Cloud Run resources.
func (r *runProxyForwarder) Stop() {
	for _, resource := range r.resources.resources {
		if resource.cancel != nil {
			if resource.cmd != nil {
				if err := resource.cmd.Process.Signal(os.Interrupt); err != nil {
					// signaling didn't work, force cancel
					resource.cancel()
				}
			} else {
				// we don't have a command, force cancel the context.
				resource.cancel()
			}
			resource.cancel = nil
			resource.cmd = nil
		}
	}
}
