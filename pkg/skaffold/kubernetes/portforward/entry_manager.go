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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
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
			entry.resource.Name)
	}
)

type forwardedPorts struct {
	ports map[int]struct{}
	lock  *sync.Mutex
}

func (f forwardedPorts) Store(key, _ interface{}) {
	f.lock.Lock()
	defer f.lock.Unlock()
	val, ok := key.(int)
	if !ok {
		panic("only store keys of type int in forwardedPorts")
	}
	// this map is only used as a set of keys, we don't care about the values
	f.ports[val] = dummy()
}

func (f forwardedPorts) LoadOrStore(key, _ interface{}) (interface{}, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	k, ok := key.(int)
	if !ok {
		return nil, false
	}
	// this map is only used as a set of keys, we don't care about the values
	_, exists := f.ports[k]
	val := dummy()
	if !exists {
		f.ports[k] = val
	}
	return val, exists
}

func dummy() struct{} {
	return struct{}{}
}

func (f forwardedPorts) Delete(port int) {
	f.lock.Lock()
	defer f.lock.Unlock()
	delete(f.ports, port)
}

type forwardedResources struct {
	resources map[string]*portForwardEntry
	lock      *sync.Mutex
}

func (f forwardedResources) Store(key, value interface{}) {
	f.lock.Lock()
	defer f.lock.Unlock()
	k, ok := key.(string)
	if !ok {
		panic("only store keys of type string in forwardedResources")
	}
	val, ok := value.(*portForwardEntry)
	if !ok {
		panic("only store values of type *portForwardEntry in forwardedResources")
	}
	f.resources[k] = val
}

func (f forwardedResources) Load(key string) (*portForwardEntry, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	val, exists := f.resources[key]
	return val, exists
}

func (f forwardedResources) Delete(resource string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	delete(f.resources, resource)
}

func (f forwardedResources) Length() int {
	f.lock.Lock()
	defer f.lock.Unlock()
	return len(f.resources)
}

func newForwardedPorts() forwardedPorts {
	return forwardedPorts{
		lock:  &sync.Mutex{},
		ports: map[int]struct{}{},
	}
}

func newForwardedResources() forwardedResources {
	return forwardedResources{
		lock:      &sync.Mutex{},
		resources: map[string]*portForwardEntry{},
	}
}

// EntryManager handles forwarding entries and keeping track of
// forwarded ports and resources.
type EntryManager struct {
	EntryForwarder
	output io.Writer

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts forwardedPorts

	// forwardedResources is a map of portForwardEntry key (string) -> portForwardEntry
	forwardedResources forwardedResources
}

// NewEntryManager returns a new port forward entry manager to keep track
// of forwarded ports and resources
func NewEntryManager(out io.Writer, cli *kubectl.CLI) EntryManager {
	return EntryManager{
		output:             out,
		forwardedPorts:     newForwardedPorts(),
		forwardedResources: newForwardedResources(),
		EntryForwarder:     &KubectlForwarder{kubectl: cli, out: out},
	}
}

func (b *EntryManager) forwardPortForwardEntry(ctx context.Context, entry *portForwardEntry) {
	// Check if this resource has already been forwarded
	if _, ok := b.forwardedResources.Load(entry.key()); ok {
		return
	}
	b.forwardedResources.Store(entry.key(), entry)

	b.Forward(ctx, entry)

	color.Green.Fprintln(
		b.output,
		fmt.Sprintf("Port forwarding %s/%s in namespace %s, remote port %d -> local port %d",
			entry.resource.Type,
			entry.resource.Name,
			entry.resource.Namespace,
			entry.resource.Port,
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
	b.EntryForwarder.Terminate(p)
}
