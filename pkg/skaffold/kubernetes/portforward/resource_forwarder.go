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
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceForwarder is responsible for forwarding user defined port forwarding resources and automatically forwarding
// services deployed by skaffold.
type ResourceForwarder struct {
	EntryManager
	userDefinedResources []*latest.PortForwardResource
	label                string
}

var (
	// For testing
	retrieveAvailablePort = util.GetAvailablePort
	retrieveServices      = retrieveServiceResources
)

// NewResourceForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewResourceForwarder(em EntryManager, label string, userDefinedResources []*latest.PortForwardResource) *ResourceForwarder {
	return &ResourceForwarder{
		EntryManager:         em,
		userDefinedResources: userDefinedResources,
		label:                label,
	}
}

// Start gets a list of services deployed by skaffold as []latest.PortForwardResource and
// forwards them.
func (p *ResourceForwarder) Start(ctx context.Context) error {
	serviceResources, err := retrieveServices(p.label)
	if err != nil {
		return errors.Wrap(err, "retrieving services for automatic port forwarding")
	}
	p.portForwardResources(ctx, append(p.userDefinedResources, serviceResources...))
	return nil
}

// Port forward each resource individually in a goroutine
func (p *ResourceForwarder) portForwardResources(ctx context.Context, resources []*latest.PortForwardResource) {
	for _, r := range resources {
		r := r
		go func() {
			p.portForwardResource(ctx, *r)
		}()
	}
}

func (p *ResourceForwarder) portForwardResource(ctx context.Context, resource latest.PortForwardResource) {
	// Get port forward entry for this resource
	entry := p.getCurrentEntry(resource)
	// Forward the entry
	p.forwardPortForwardEntry(ctx, entry)
}

func (p *ResourceForwarder) getCurrentEntry(resource latest.PortForwardResource) *portForwardEntry {
	// determine if we have seen this before
	entry := &portForwardEntry{
		resource:        resource,
		terminationLock: &sync.Mutex{},
	}
	// If we have, return the current entry
	oldEntry, ok := p.forwardedResources.Load(entry.key())

	if ok {
		entry.localPort = oldEntry.localPort
		return entry
	}

	// retrieve an open port on the host
	entry.localPort = retrieveAvailablePort(resource.LocalPort, &p.forwardedPorts)
	return entry
}

// retrieveServiceResources retrieves all services in the cluster matching the given label
// as a list of PortForwardResources
func retrieveServiceResources(label string) ([]*latest.PortForwardResource, error) {
	clientset, err := kubernetes.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting clientset")
	}
	services, err := clientset.CoreV1().Services("").List(metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "selecting services by label %s", label)
	}
	var resources []*latest.PortForwardResource
	for _, s := range services.Items {
		for _, p := range s.Spec.Ports {
			resources = append(resources, &latest.PortForwardResource{
				Type:      constants.Service,
				Name:      s.Name,
				Namespace: s.Namespace,
				Port:      int(p.Port),
				LocalPort: int(p.Port),
			})
		}
	}
	return resources, nil
}
