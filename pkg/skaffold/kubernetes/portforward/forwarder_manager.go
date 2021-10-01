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

	"golang.org/x/sync/singleflight"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

type Config interface {
	kubectl.Config

	Mode() config.RunMode
	PortForwardResources() []*latestV2.PortForwardResource
	PortForwardOptions() config.PortForwardOptions
}

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	// Start initiates the forwarder's operation. It should not return until any ports have been allocated.
	Start(ctx context.Context, out io.Writer, namespaces []string) error
	Stop()
}

// ForwarderManager manages all forwarders
type ForwarderManager struct {
	forwarders   []Forwarder
	entryManager *EntryManager
	label        string

	singleRun  singleflight.Group
	namespaces *[]string
}

// NewForwarderManager returns a new port manager which handles starting and stopping port forwarding
func NewForwarderManager(cli *kubectl.CLI, podSelector kubernetes.PodSelector, label string, runMode config.RunMode, namespaces *[]string,
	options config.PortForwardOptions, userDefined []*latestV2.PortForwardResource) *ForwarderManager {
	if !options.Enabled() {
		return nil
	}

	entryManager := NewEntryManager(NewKubectlForwarder(cli))

	// The order matters to ensure user-defined port-forwards with local-ports are processed first.
	var forwarders []Forwarder
	if options.ForwardUser(runMode) {
		forwarders = append(forwarders, NewUserDefinedForwarder(entryManager, cli.KubeContext, userDefined))
	}
	if options.ForwardServices(runMode) {
		forwarders = append(forwarders, NewServicesForwarder(entryManager, cli.KubeContext, label))
	}
	if options.ForwardPods(runMode) {
		forwarders = append(forwarders, NewWatchingPodForwarder(entryManager, cli.KubeContext, podSelector, allPorts))
	} else if options.ForwardDebug(runMode) {
		forwarders = append(forwarders, NewWatchingPodForwarder(entryManager, cli.KubeContext, podSelector, debugPorts))
	}

	return &ForwarderManager{
		forwarders:   forwarders,
		entryManager: entryManager,
		label:        label,
		singleRun:    singleflight.Group{},
		namespaces:   namespaces,
	}
}

func allPorts(pod *v1.Pod, c v1.Container) []v1.ContainerPort {
	return c.Ports
}

func debugPorts(pod *v1.Pod, c v1.Container) []v1.ContainerPort {
	var ports []v1.ContainerPort

	annot, found := pod.ObjectMeta.Annotations[types.DebugConfig]
	if !found {
		return nil
	}
	var configurations map[string]types.ContainerDebugConfiguration
	if err := json.Unmarshal([]byte(annot), &configurations); err != nil {
		log.Entry(context.TODO()).Warnf("could not decode debug annotation on pod/%s (%q): %v", pod.Name, annot, err)
		return nil
	}
	dc, found := configurations[c.Name]
	if !found {
		log.Entry(context.TODO()).Debugf("no debug configuration found on pod/%s/%s", pod.Name, c.Name)
		return nil
	}
	for _, port := range c.Ports {
		for _, exposed := range dc.Ports {
			if uint32(port.ContainerPort) == exposed {
				log.Entry(context.TODO()).Debugf("selecting debug port for pod/%s/%s: %v", pod.Name, c.Name, port)
				ports = append(ports, port)
			}
		}
	}
	return ports
}

// Start begins all forwarders managed by the ForwarderManager
func (p *ForwarderManager) Start(ctx context.Context, out io.Writer) error {
	// Port forwarding is not enabled.
	if p == nil {
		return nil
	}

	_, err, _ := p.singleRun.Do(p.label, func() (interface{}, error) {
		return struct{}{}, p.start(ctx, out)
	})
	return err
}

// Start begins all forwarders managed by the ForwarderManager
func (p *ForwarderManager) start(ctx context.Context, out io.Writer) error {
	eventV2.TaskInProgress(constants.PortForward, "Port forward URLs")
	ctx, endTrace := instrumentation.StartTrace(ctx, "Start")
	defer endTrace()

	p.entryManager.Start(out)
	for _, f := range p.forwarders {
		if err := f.Start(ctx, out, *p.namespaces); err != nil {
			eventV2.TaskFailed(constants.PortForward, err)
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
	}

	eventV2.TaskSucceeded(constants.PortForward)
	return nil
}

func (p *ForwarderManager) Stop() {
	// Port forwarding is not enabled.
	if p == nil {
		return
	}
	p.singleRun.Do(p.label, func() (interface{}, error) {
		p.stop()
		return struct{}{}, nil
	})
}

// Stop cleans up and terminates all forwarders managed by the ForwarderManager
func (p *ForwarderManager) stop() {
	for _, f := range p.forwarders {
		f.Stop()
	}
}

func (p *ForwarderManager) Name() string {
	return "PortForwarding"
}

func (p *ForwarderManager) AddPodForwarder(cli *kubectl.CLI, podSelector kubernetes.PodSelector, runMode config.RunMode, options config.PortForwardOptions) {
	if options.ForwardPods(runMode) {
		p.forwarders = append(p.forwarders, NewWatchingPodForwarder(p.entryManager, cli.KubeContext, podSelector, allPorts))
	} else if options.ForwardDebug(runMode) {
		p.forwarders = append(p.forwarders, NewWatchingPodForwarder(p.entryManager, cli.KubeContext, podSelector, debugPorts))
	}
}
