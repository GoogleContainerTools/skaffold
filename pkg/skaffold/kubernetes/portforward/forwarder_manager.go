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

	"github.com/sirupsen/logrus"

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
func NewForwarderManager(out io.Writer, cli *kubectl.CLI, podSelector kubernetes.PodSelector, namespaces []string, label string, options config.PortForwardOptions, userDefined []*latest.PortForwardResource) *ForwarderManager {
	logrus.Warnf(">>> port-forwarding for %+v", options)
	entryManager := NewEntryManager(out, NewKubectlForwarder(out, cli))

	var forwarders []Forwarder
	var forwardUser, forwardDebug, forwardServices, forwardPods bool
	for _, o := range options.Modes {
		switch o {
		case "none", "false":
			return nil
		case "true":
			forwardUser = true
			forwardServices = true
		case "user":
			forwardUser = true
		case "services":
			forwardServices = true
		case "pods":
			forwardPods = true
		case "debug":
			forwardDebug = true
		default:
			logrus.Warn("Unknown port-forwarding option: %q", o)
		}
	}
	if forwardUser {
		forwarders = append(forwarders, NewUserDefinedForwarder(entryManager, namespaces, userDefined))
	}
	if forwardServices {
		forwarders = append(forwarders, NewServicesForwarder(entryManager, namespaces, label))
	}
	if forwardPods {
		forwarders = append(forwarders, NewWatchingPodForwarder(entryManager, podSelector, namespaces))
	} else if forwardDebug {
		// TODO: just forward debug-related ports
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
