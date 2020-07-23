/*
Copyright 2019 The Skaffold Authors

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

package portforward

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// ResourceForwarder is responsible for forwarding user defined port forwarding resources and automatically forwarding
// services deployed by skaffold.
type ResourceForwarder struct {
	entryManager         *EntryManager
	namespaces           []string
	runID                string
	userDefinedResources []*latest.PortForwardResource
}

var (
	// For testing
	retrieveAvailablePort = util.GetAvailablePort
	retrieveServices      = retrieveServiceResources
)

// NewResourceForwarder returns a struct that tracks and port-forwards services as they are created and modified
func NewResourceForwarder(entryManager *EntryManager, namespaces []string, runID string, userDefinedResources []*latest.PortForwardResource) *ResourceForwarder {
	return &ResourceForwarder{
		entryManager:         entryManager,
		namespaces:           namespaces,
		runID:                runID,
		userDefinedResources: userDefinedResources,
	}
}

// Start gets a list of services deployed by skaffold as []latest.PortForwardResource and
// forwards them.
func (p *ResourceForwarder) Start(ctx context.Context) error {
	serviceResources, err := retrieveServices(p.runID, p.namespaces)
	if err != nil {
		return fmt.Errorf("retrieving services for automatic port forwarding: %w", err)
	}
	p.portForwardResources(ctx, append(p.userDefinedResources, serviceResources...))
	return nil
}

func (p *ResourceForwarder) Stop() {
	p.entryManager.Stop()
}

// Port forward each resource individually in a goroutine
func (p *ResourceForwarder) portForwardResources(ctx context.Context, resources []*latest.PortForwardResource) {
	go func() {
		for _, r := range resources {
			p.portForwardResource(ctx, *r)
		}
	}()
}

func (p *ResourceForwarder) portForwardResource(ctx context.Context, resource latest.PortForwardResource) {
	// Get port forward entry for this resource
	entry := p.getCurrentEntry(resource)
	// Forward the entry
	p.entryManager.forwardPortForwardEntry(ctx, entry)
}

func (p *ResourceForwarder) getCurrentEntry(resource latest.PortForwardResource) *portForwardEntry {
	// determine if we have seen this before
	entry := newPortForwardEntry(0, resource, "", "", "", "", 0, false)

	// If we have, return the current entry
	oldEntry, ok := p.entryManager.forwardedResources.Load(entry.key())
	if ok {
		entry.localPort = oldEntry.localPort
		return entry
	}

	// retrieve an open port on the host
	entry.localPort = retrieveAvailablePort(resource.Address, resource.LocalPort, &p.entryManager.forwardedPorts)
	return entry
}

// retrieveServiceResources retrieves all services in the cluster matching the given label
// as a list of PortForwardResources
func retrieveServiceResources(runID string, namespaces []string) ([]*latest.PortForwardResource, error) {
	client, err := kubernetesclient.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	var resources []*latest.PortForwardResource
	for _, ns := range namespaces {
		unfilteredServices, err := client.CoreV1().Services(ns).List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("retrieving all services: %w", err)
		}

		// only select services with matching run-id annotation
		var services []v1.Service
		for _, s := range unfilteredServices.Items {
			if kubernetes.HasRunIDAnnotation(s.GetAnnotations(), runID) {
				services = append(services, s)
			}
		}

		for _, s := range services {
			for _, p := range s.Spec.Ports {
				resources = append(resources, &latest.PortForwardResource{
					Type:      constants.Service,
					Name:      s.Name,
					Namespace: s.Namespace,
					Port:      int(p.Port),
					Address:   constants.DefaultPortForwardAddress,
					LocalPort: int(p.Port),
				})
			}
		}
	}
	return resources, nil
}
