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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Start(ctx context.Context) error
	Stop()
}

// ForwarderManager manages all forwarders
type ForwarderManager struct {
	forwarders []Forwarder
}

// NewForwarderManager returns a new port manager which handles starting and stopping port forwarding
func NewForwarderManager(out io.Writer, cli *kubectl.CLI, podSelector kubernetes.PodSelector, namespaces []string, label string, opts config.PortForwardOptions, userDefined []*latest.PortForwardResource) *ForwarderManager {
	entryManager := NewEntryManager(out, NewKubectlForwarder(out, cli))

	var forwarders []Forwarder
	forwarders = append(forwarders, NewResourceForwarder(entryManager, namespaces, label, userDefined))
	if opts.ForwardPods {
		forwarders = append(forwarders, NewWatchingPodForwarder(entryManager, podSelector, namespaces))
	}

	return &ForwarderManager{
		forwarders: forwarders,
	}
}

// Start begins all forwarders managed by the ForwarderManager
func (p *ForwarderManager) Start(ctx context.Context) error {
	// Port forwarding is not enabled.
	if p == nil {
		return nil
	}

	for _, f := range p.forwarders {
		if err := f.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop cleans up and terminates all forwarders managed by the ForwarderManager
func (p *ForwarderManager) Stop() {
	// Port forwarding is not enabled.
	if p == nil {
		return
	}

	for _, f := range p.forwarders {
		f.Stop()
	}
}
