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
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type portForwardEntry struct {
	resourceVersion        int
	resource               latest.PortForwardResource
	podName                string
	containerName          string
	portName               string
	ownerReference         string
	localPort              int
	automaticPodForwarding bool
	terminated             bool
	terminationLock        sync.Mutex
	cancel                 context.CancelFunc
}

// newPortForwardEntry returns a port forward entry.
func newPortForwardEntry(resourceVersion int, resource latest.PortForwardResource, podName, containerName, portName, ownerReference string, localPort int, automaticPodForwarding bool) *portForwardEntry {
	return &portForwardEntry{
		resourceVersion:        resourceVersion,
		resource:               resource,
		podName:                podName,
		containerName:          containerName,
		portName:               portName,
		ownerReference:         ownerReference,
		localPort:              localPort,
		automaticPodForwarding: automaticPodForwarding,
	}
}

// key is an identifier for the lock on a port during the skaffold dev cycle.
// if automaticPodForwarding is set, we return a key that doesn't include podName, since we want the key
// to be the same whenever pods restart
func (p *portForwardEntry) key() string {
	if p.automaticPodForwarding {
		return fmt.Sprintf("%s-%s-%s-%s-%d", p.ownerReference, p.containerName, p.resource.Namespace, p.portName, p.resource.Port)
	}
	return fmt.Sprintf("%s-%s-%s-%d", strings.ToLower(string(p.resource.Type)), p.resource.Name, p.resource.Namespace, p.resource.Port)
}

// String is a utility function that returns the port forward entry as a user-readable string
func (p *portForwardEntry) String() string {
	return fmt.Sprintf("%s-%s-%s-%d", strings.ToLower(string(p.resource.Type)), p.resource.Name, p.resource.Namespace, p.resource.Port)
}
