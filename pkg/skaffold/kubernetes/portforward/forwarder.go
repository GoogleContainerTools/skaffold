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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	// For testing
	forwardingPollTime = time.Minute
)

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Start(ctx context.Context) error
	Stop()
}

// GetForwarders returns a list of forwarders
func GetForwarders(out io.Writer, podSelector kubernetes.PodSelector, namespaces []string, label string, automaticPodForwarding bool) []Forwarder {
	baseForwarder := NewBaseForwarder(out, namespaces)
	var f []Forwarder
	rf := NewResourceForwarder(baseForwarder, label)
	f = append(f, rf)

	if automaticPodForwarding {
		apf := NewAutomaticPodForwarder(baseForwarder, podSelector)
		f = append(f, apf)
	}
	return f
}

// BaseForwarder is the base port forwarder for automatic port forwarding
// and for port forwarding generic resources
type BaseForwarder struct {
	PortForwardEntryForwarder
	output     io.Writer
	namespaces []string

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts *sync.Map

	// forwardedResources is a map of portForwardEntry key (string) -> portForwardEntry
	forwardedResources map[string]*portForwardEntry
}

func NewBaseForwarder(out io.Writer, namespaces []string) BaseForwarder {
	return BaseForwarder{
		output:                    out,
		namespaces:                namespaces,
		forwardedPorts:            &sync.Map{},
		forwardedResources:        make(map[string]*portForwardEntry),
		PortForwardEntryForwarder: &KubectlForwarder{},
	}
}

func (b *BaseForwarder) forwardPortForwardEntry(ctx context.Context, entry *portForwardEntry) error {
	b.forwardedResources[entry.key()] = entry
	color.Default.Fprintln(b.output, fmt.Sprintf("Port Forwarding %s/%s %d -> %d", entry.resource.Type, entry.resource.Name, entry.resource.Port, entry.localPort))
	return wait.PollImmediate(time.Second, forwardingPollTime, func() (bool, error) {
		if err := b.Forward(ctx, entry); err != nil {
			return false, nil
		}
		return true, nil
	})
}

// Stop terminates all kubectl port-forward commands.
func (b *BaseForwarder) Stop() {
	for _, entry := range b.forwardedResources {
		b.Terminate(entry)
	}
}
