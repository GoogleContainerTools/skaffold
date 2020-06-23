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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	portForwardEvent = func(entry *portForwardEntry) {
		// TODO priyawadhwa@, change event API to accept ports of type int
		event.PortForwarded(
			int32(entry.localPort),
			int32(entry.resource.Port),
			entry.podName,
			entry.containerName,
			entry.resource.Namespace,
			entry.portName,
			string(entry.resource.Type),
			entry.resource.Name,
			entry.resource.Address)
	}
)

type forwardedResources struct {
	resources map[string]*portForwardEntry
	lock      sync.Mutex
}

func (f *forwardedResources) Store(k string, v *portForwardEntry) {
	f.lock.Lock()

	if f.resources == nil {
		f.resources = map[string]*portForwardEntry{}
	}
	f.resources[k] = v

	f.lock.Unlock()
}

func (f *forwardedResources) Load(key string) (*portForwardEntry, bool) {
	f.lock.Lock()
	val, exists := f.resources[key]
	f.lock.Unlock()

	return val, exists
}

func (f *forwardedResources) Delete(key string) {
	f.lock.Lock()
	delete(f.resources, key)
	f.lock.Unlock()
}

func (f *forwardedResources) Length() int {
	f.lock.Lock()
	length := len(f.resources)
	f.lock.Unlock()

	return length
}

// EntryManager handles forwarding entries and keeping track of
// forwarded ports and resources.
type EntryManager struct {
	output         io.Writer
	entryForwarder EntryForwarder

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts util.PortSet

	// forwardedResources is a map of portForwardEntry key (string) -> portForwardEntry
	forwardedResources forwardedResources
}

// NewEntryManager returns a new port forward entry manager to keep track
// of forwarded ports and resources
func NewEntryManager(out io.Writer, entryForwarder EntryForwarder) *EntryManager {
	return &EntryManager{
		output:         out,
		entryForwarder: entryForwarder,
	}
}

func (b *EntryManager) forwardPortForwardEntry(ctx context.Context, entry *portForwardEntry) {
	// Check if this resource has already been forwarded
	if _, ok := b.forwardedResources.Load(entry.key()); ok {
		return
	}
	b.forwardedResources.Store(entry.key(), entry)

	b.entryForwarder.Forward(ctx, entry)

	color.Green.Fprintln(
		b.output,
		fmt.Sprintf("Port forwarding %s/%s in namespace %s, remote port %d -> address %s port %d",
			entry.resource.Type,
			entry.resource.Name,
			entry.resource.Namespace,
			entry.resource.Port,
			entry.resource.Address,
			entry.localPort))
	portForwardEvent(entry)
}

// Stop terminates all kubectl port-forward commands.
func (b *EntryManager) Stop() {
	for _, pfe := range b.forwardedResources.resources {
		b.Terminate(pfe)
	}
}

// Terminate terminates a single port forward entry
func (b *EntryManager) Terminate(p *portForwardEntry) {
	b.forwardedResources.Delete(p.key())
	b.forwardedPorts.Delete(p.localPort)
	b.entryForwarder.Terminate(p)
}
