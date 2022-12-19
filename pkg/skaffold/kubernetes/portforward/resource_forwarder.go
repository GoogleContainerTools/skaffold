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
	"io"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// ResourceForwarder is responsible for forwarding user defined port forwarding resources and automatically forwarding
// services deployed by skaffold.
type ResourceForwarder struct {
	output               io.Writer
	entryManager         *EntryManager
	label                string
	userDefinedResources []*latest.PortForwardResource
	kubeContext          string
	services             bool
}

var (
	// For testing
	retrieveAvailablePort = util.GetAvailablePort
	retrieveServices      = retrieveServiceResources
)

// NewServicesForwarder returns a struct that tracks and port-forwards services as they are created and modified
func NewServicesForwarder(entryManager *EntryManager, kubeContext string, label string) *ResourceForwarder {
	return &ResourceForwarder{
		entryManager: entryManager,
		label:        label,
		services:     true,
		kubeContext:  kubeContext,
	}
}

// NewUserDefinedForwarder returns a struct that tracks and port-forwards services as they are created and modified
func NewUserDefinedForwarder(entryManager *EntryManager, kubeContext string, userDefinedResources []*latest.PortForwardResource) *ResourceForwarder {
	return &ResourceForwarder{
		entryManager:         entryManager,
		kubeContext:          kubeContext,
		userDefinedResources: userDefinedResources,
	}
}

// Start gets a list of services deployed by skaffold as []latest.PortForwardResource and
// forwards them.
func (p *ResourceForwarder) Start(ctx context.Context, out io.Writer, namespaces []string) error {
	p.output = out
	if len(namespaces) == 1 {
		for _, pf := range p.userDefinedResources {
			if err := applyWithTemplate(pf); err != nil {
				return err
			}
			if pf.Namespace == "" {
				pf.Namespace = namespaces[0]
			}
		}
	} else {
		var validResources []*latest.PortForwardResource
		for _, pf := range p.userDefinedResources {
			if pf.Namespace != "" {
				if err := applyWithTemplate(pf); err != nil {
					return err
				}
				validResources = append(validResources, pf)
			} else {
				log.Entry(ctx).Warnf("Skipping the port forwarding resource %s/%s because namespace is not specified", pf.Type, pf.Name)
			}
		}
		p.userDefinedResources = validResources
	}

	var serviceResources []*latest.PortForwardResource
	if p.services {
		found, err := retrieveServices(ctx, p.label, namespaces, p.kubeContext)
		if err != nil {
			return fmt.Errorf("retrieving services for automatic port forwarding: %w", err)
		}
		serviceResources = found
	}
	p.portForwardResources(ctx, append(p.userDefinedResources, serviceResources...))
	return nil
}

func applyWithTemplate(resource *latest.PortForwardResource) error {
	if resource.Namespace != "" {
		namespace, err := util.ExpandEnvTemplateOrFail(resource.Namespace, nil)
		if err != nil {
			return fmt.Errorf("cannot parse the namespace template on user defined port forwarder: %w", err)
		}
		resource.Namespace = namespace
	}
	name, err := util.ExpandEnvTemplateOrFail(resource.Name, nil)
	if err != nil {
		return fmt.Errorf("cannot parse the name template on user defined port forwarder: %w", err)
	}
	resource.Name = name
	return nil
}

func (p *ResourceForwarder) Stop() {
	p.entryManager.Stop()
}

// Port forward each resource individually in a goroutine
func (p *ResourceForwarder) portForwardResources(ctx context.Context, resources []*latest.PortForwardResource) {
	var wg sync.WaitGroup
	for _, r := range resources {
		wg.Add(1)
		go func(r latest.PortForwardResource) {
			defer wg.Done()
			p.portForwardResource(ctx, r)
		}(*r)
	}
	wg.Wait()
}

func (p *ResourceForwarder) portForwardResource(ctx context.Context, resource latest.PortForwardResource) {
	// Get port forward entry for this resource
	entry := p.getCurrentEntry(resource)
	// Forward the entry
	p.entryManager.forwardPortForwardEntry(ctx, p.output, entry)
}

func (p *ResourceForwarder) getCurrentEntry(resource latest.PortForwardResource) *portForwardEntry {
	// determine if we have seen this before
	entry := newPortForwardEntry(0, resource, "", "", "", "", 0, false)

	// If we have, return the current entry
	oe, ok := p.entryManager.forwardedResources.Load(entry.key())
	if ok {
		oldEntry := oe.(*portForwardEntry)
		entry.localPort = oldEntry.localPort
		return entry
	}

	// Try to request matching local port *providing* that it is not a system port.
	// https://github.com/GoogleContainerTools/skaffold/pull/5554#issuecomment-803270340
	requestPort := resource.LocalPort
	if requestPort == 0 && resource.Port.IntVal >= 1024 {
		requestPort = resource.Port.IntVal
	}
	entry.localPort = retrieveAvailablePort(resource.Address, requestPort, &p.entryManager.forwardedPorts)
	return entry
}

// retrieveServiceResources retrieves all services in the cluster matching the given label
// as a list of PortForwardResources
func retrieveServiceResources(ctx context.Context, label string, namespaces []string, kubeContext string) ([]*latest.PortForwardResource, error) {
	client, err := kubernetesclient.Client(kubeContext)
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	var resources []*latest.PortForwardResource
	for _, ns := range namespaces {
		services, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{
			LabelSelector: label,
		})
		if err != nil {
			return nil, fmt.Errorf("selecting services by label %q: %w", label, err)
		}
		for _, s := range services.Items {
			for _, p := range s.Spec.Ports {
				resources = append(resources, &latest.PortForwardResource{
					Type:      constants.Service,
					Name:      s.Name,
					Namespace: s.Namespace,
					Port:      schemautil.FromInt(int(p.Port)),
					Address:   constants.DefaultPortForwardAddress,
				})
			}
		}
	}
	return resources, nil
}
