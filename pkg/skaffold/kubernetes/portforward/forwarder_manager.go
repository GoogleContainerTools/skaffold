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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/annotations"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

type Config interface {
	kubectl.Config

	Mode() config.RunMode
	PortForwardResources() []*latestV1.PortForwardResource
	PortForwardOptions() config.PortForwardOptions
}

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Start(ctx context.Context, out io.Writer, namespaces []string) error
	Stop()
}

// ForwarderManager manages all forwarders
type ForwarderManager struct {
	forwarders   []Forwarder
	entryManager *EntryManager
}

// NewForwarderManager returns a new port manager which handles starting and stopping port forwarding
func NewForwarderManager(cli *kubectl.CLI, podSelector kubernetes.PodSelector, label string, runMode config.RunMode, options config.PortForwardOptions, userDefined []*latestV1.PortForwardResource) *ForwarderManager {
	if !options.Enabled() {
		return nil
	}

	entryManager := NewEntryManager(NewKubectlForwarder(cli))

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
		forwarders:   forwarders,
		entryManager: entryManager,
	}
}

func allPorts(pod *v1.Pod, c v1.Container) []v1.ContainerPort {
	return c.Ports
}

func debugPorts(pod *v1.Pod, c v1.Container) []v1.ContainerPort {
	var ports []v1.ContainerPort

	annot, found := pod.ObjectMeta.Annotations[annotations.DebugConfig]
	if !found {
		return nil
	}
	var configurations map[string]annotations.ContainerDebugConfiguration
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
func (p *ForwarderManager) Start(ctx context.Context, out io.Writer, namespaces []string) error {
	// Port forwarding is not enabled.
	if p == nil {
		return nil
	}

	eventV2.TaskInProgress(constants.PortForward, "Port forward URLs")
	ctx, endTrace := instrumentation.StartTrace(ctx, "Start")
	defer endTrace()

	p.entryManager.Start(out)
	for _, f := range p.forwarders {
		if err := f.Start(ctx, out, namespaces); err != nil {
			eventV2.TaskFailed(constants.PortForward, err)
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
	}

	eventV2.TaskSucceeded(constants.PortForward)
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

func (p *ForwarderManager) Name() string {
	return "PortForwarding"
}
