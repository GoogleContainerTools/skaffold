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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	portForwardEvent = func(entry *portForwardEntry) {
		event.PortForwarded(
			int32(entry.localPort),
			entry.resource.Port,
			entry.podName,
			entry.containerName,
			entry.resource.Namespace,
			entry.portName,
			string(entry.resource.Type),
			entry.resource.Name,
			entry.resource.Address)
	}
	portForwardEventV2 = func(entry *portForwardEntry) {
		eventV2.PortForwarded(
			int32(entry.localPort),
			entry.resource.Port,
			entry.podName,
			entry.containerName,
			entry.resource.Namespace,
			entry.portName,
			string(entry.resource.Type),
			entry.resource.Name,
			entry.resource.Address)
	}
)

// EntryManager handles forwarding entries and keeping track of
// forwarded ports and resources.
type EntryManager struct {
	entryForwarder EntryForwarder

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts util.PortSet

	// forwardedResources is a map of portForwardEntry key (string) -> portForwardEntry
	forwardedResources sync.Map
}

// NewEntryManager returns a new port forward entry manager to keep track
// of forwarded ports and resources
func NewEntryManager(entryForwarder EntryForwarder) *EntryManager {
	return &EntryManager{
		entryForwarder: entryForwarder,
	}
}

func (b *EntryManager) forwardPortForwardEntry(ctx context.Context, out io.Writer, entry *portForwardEntry) {
	out = output.WithEventContext(out, constants.PortForward, fmt.Sprintf("%s/%s", entry.resource.Type, entry.resource.Name))

	// Check if this resource has already been forwarded
	if _, found := b.forwardedResources.LoadOrStore(entry.key(), entry); found {
		return
	}

	if err := b.entryForwarder.Forward(ctx, entry); err == nil {
		output.Green.Fprintln(
			out,
			fmt.Sprintf("Port forwarding %s/%s in namespace %s, remote port %s -> http://%s:%d",
				entry.resource.Type,
				entry.resource.Name,
				entry.resource.Namespace,
				entry.resource.Port.String(),
				entry.resource.Address,
				entry.localPort))
	} else {
		output.Red.Fprintln(out, err)
	}
	portForwardEvent(entry)
	portForwardEventV2(entry)
}

// Start ensures the underlying entryForwarder is ready to forward.
func (b *EntryManager) Start(out io.Writer) {
	b.entryForwarder.Start(out)
}

// Stop terminates all kubectl port-forward commands.
func (b *EntryManager) Stop() {
	b.forwardedResources.Range(func(_, value interface{}) bool {
		pfe := value.(*portForwardEntry)
		b.Terminate(pfe)
		return true
	})
}

// Terminate terminates a single port forward entry
func (b *EntryManager) Terminate(p *portForwardEntry) {
	b.forwardedResources.Delete(p.key())
	b.forwardedPorts.Delete(p.localPort)
	b.entryForwarder.Terminate(p)
}
