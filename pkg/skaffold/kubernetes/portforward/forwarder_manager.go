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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Start(ctx context.Context) error
	Stop()
}

// ForwarderManager manages all forwarders
type ForwarderManager struct {
	output io.Writer

	EntryForwarder
	Forwarders []Forwarder
}

var (
	emptyForwarderManager = &ForwarderManager{}
)

// NewForwarderManager returns a new port manager which handles starting and stopping port forwarding
func NewForwarderManager(out io.Writer, podSelector kubernetes.PodSelector, namespaces []string, label string, opts config.PortForwardOptions) *ForwarderManager {
	if !opts.Enabled {
		return emptyForwarderManager
	}

	em := NewEntryManager(out)

	ForwarderManager := &ForwarderManager{
		output:     out,
		Forwarders: []Forwarder{NewResourceForwarder(em, label)},
	}

	if opts.ForwardPods {
		f := NewWatchingPodForwarder(em, podSelector, namespaces)
		ForwarderManager.Forwarders = append(ForwarderManager.Forwarders, f)
	}

	return ForwarderManager
}

// Start begins all forwarders managed by the ForwarderManager
func (p *ForwarderManager) Start(ctx context.Context) error {
	for _, f := range p.Forwarders {
		if err := f.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop cleans up and terminates all forwarders managed by the ForwarderManager
func (p *ForwarderManager) Stop() {
	for _, f := range p.Forwarders {
		f.Stop()
	}
}
