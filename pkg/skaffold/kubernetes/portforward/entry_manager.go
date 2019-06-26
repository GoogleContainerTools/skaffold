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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	// For testing
	forwardingTimeoutTime = time.Minute
)

// EntryManager handles forwarding entries and keeping track of
// forwarded ports and resources.
type EntryManager struct {
	EntryForwarder
	output io.Writer

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts *sync.Map

	// forwardedResources is a map of portForwardEntry key (string) -> portForwardEntry
	forwardedResources *sync.Map
}

// NewEntryManager returns a new port forward entry manager to keep track
// of forwarded ports and resources
func NewEntryManager(out io.Writer) EntryManager {
	return EntryManager{
		output:             out,
		forwardedPorts:     &sync.Map{},
		forwardedResources: &sync.Map{},
		EntryForwarder:     &KubectlForwarder{},
	}
}

func (b *EntryManager) forwardPortForwardEntry(ctx context.Context, entry *portForwardEntry) error {
	b.forwardedResources.Store(entry.key(), entry)
	color.Default.Fprintln(b.output, fmt.Sprintf("Port Forwarding %s/%s %d -> %d", entry.resource.Type, entry.resource.Name, entry.resource.Port, entry.localPort))
	err := wait.PollImmediate(time.Second, forwardingTimeoutTime, func() (bool, error) {
		if err := b.Forward(ctx, entry); err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	event.PortForwarded(entry.localPort, entry.resource.Port, entry.podName, entry.containerName, entry.resource.Namespace, entry.portName, string(entry.resource.Type), entry.resource.Name)
	return nil
}

// Stop terminates all kubectl port-forward commands.
func (b *EntryManager) Stop() {
	b.forwardedResources.Range(func(key, value interface{}) bool {
		entry := value.(*portForwardEntry)
		b.Terminate(entry)
		return true
	})
}
