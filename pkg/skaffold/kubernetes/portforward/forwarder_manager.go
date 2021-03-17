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
	"encoding/json"
	"io"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	debugging "github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Start(ctx context.Context, namespaces []string) error
	Stop()
}

// ForwarderManager manages all forwarders
type ForwarderManager struct {
	forwarders []Forwarder
}

// NewForwarderManager returns a new port manager which handles starting and stopping port forwarding
func NewForwarderManager(out io.Writer, cli *kubectl.CLI, podSelector kubernetes.PodSelector, namespaces []string, label string, runMode config.RunMode, options config.PortForwardOptions, userDefined []*latest.PortForwardResource) *ForwarderManager {
	if !options.Enabled() {
		return nil
	}

	// TODO this doesn't feel like the right place
	if err := options.Validate(); err != nil {
		logrus.Error("port-forward: ", err)
		return nil
	}

	entryManager := NewEntryManager(out, NewKubectlForwarder(out, cli))

	var forwarders []Forwarder
	if options.ForwardUser(runMode) {
		forwarders = append(forwarders, NewUserDefinedForwarder(entryManager, userDefined))
	}
	if options.ForwardServices(runMode) {
		forwarders = append(forwarders, NewServicesForwarder(entryManager, label))
	}
	if options.ForwardPods(runMode) {
		forwarders = append(forwarders, NewWatchingPodForwarder(entryManager, podSelector, allPorts))
	} else if options.ForwardDebug(runMode) {
		forwarders = append(forwarders, NewWatchingPodForwarder(entryManager, podSelector, debugPorts))
	}

	return &ForwarderManager{
		forwarders: forwarders,
	}
}

func allPorts(pod *v1.Pod, c v1.Container) []v1.ContainerPort {
	return c.Ports
}

func debugPorts(pod *v1.Pod, c v1.Container) []v1.ContainerPort {
	var ports []v1.ContainerPort

	annot, found := pod.ObjectMeta.Annotations[debugging.DebugConfigAnnotation]
	if !found {
		return nil
	}
	var configurations map[string]debugging.ContainerDebugConfiguration
	if err := json.Unmarshal([]byte(annot), &configurations); err != nil {
		logrus.Warnf("could not decode debug annotation on pod/%s (%q): %v", pod.Name, annot, err)
		return nil
	}
	dc, found := configurations[c.Name]
	if !found {
		logrus.Debugf("no debug configuration found on pod/%s/%s", pod.Name, c.Name)
		return nil
	}
	for _, port := range c.Ports {
		for _, exposed := range dc.Ports {
			if uint32(port.ContainerPort) == exposed {
				logrus.Debugf("selecting debug port for pod/%s/%s: %v", pod.Name, c.Name, port)
				ports = append(ports, port)
			}
		}
	}
	return ports
}

// Start begins all forwarders managed by the ForwarderManager
func (p *ForwarderManager) Start(ctx context.Context, namespaces []string) error {
	// Port forwarding is not enabled.
	if p == nil {
		return nil
	}

	for _, f := range p.forwarders {
		if err := f.Start(ctx, namespaces); err != nil {
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
