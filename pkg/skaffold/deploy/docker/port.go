/*
Copyright 2021 The Skaffold Authors

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

package docker

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type PortManager struct {
	containerPorts map[string][]int // maps containers to the ports they have allocated
	portSet        util.PortSet

	lock sync.Mutex
}

func NewPortManager() *PortManager {
	return &PortManager{
		containerPorts: make(map[string][]int),
		portSet:        util.PortSet{},
	}
}

// getPorts converts PortForwardResources into docker.PortSet/PortMap objects.
// These are passed to ContainerCreate on Deploy to expose container ports on the host.
// It also returns a list of containerPortForwardEntry, to be passed to the event handler
func (pm *PortManager) getPorts(containerName string, pf []*v1.PortForwardResource) (nat.PortSet, nat.PortMap, []containerPortForwardEntry, error) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	s := make(nat.PortSet)
	m := make(nat.PortMap)
	var entries []containerPortForwardEntry
	var ports []int
	for _, p := range pf {
		if strings.ToLower(string(p.Type)) != "container" {
			logrus.Debugf("skipping non-container port forward resource in Docker deploy: %s\n", p.Name)
			continue
		}
		localPort := util.GetAvailablePort(p.Address, p.LocalPort, &pm.portSet)
		ports = append(ports, localPort)
		port, err := nat.NewPort("tcp", p.Port.String())
		if err != nil {
			return nil, nil, nil, err
		}
		s[port] = struct{}{}
		m[port] = []nat.PortBinding{
			{HostIP: p.Address, HostPort: fmt.Sprintf("%d", localPort)},
		}
		entries = append(entries, containerPortForwardEntry{
			container:       containerName,
			resourceName:    p.Name,
			resourceAddress: p.Address,
			localPort:       int32(localPort),
			remotePort:      p.Port,
		})
	}
	pm.containerPorts[containerName] = ports
	return s, m, entries, nil
}

func (pm *PortManager) relinquishPorts(containerName string) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	ports := pm.containerPorts[containerName]
	for _, port := range ports {
		pm.portSet.Delete(port)
	}
	pm.containerPorts[containerName] = nil
}

type containerPortForwardEntry struct {
	container       string
	resourceName    string
	resourceAddress string
	localPort       int32
	remotePort      schemautil.IntOrString
}

func containerPortForwardEvents(out io.Writer, entries []containerPortForwardEntry) {
	for _, entry := range entries {
		event.PortForwarded(
			entry.localPort,
			entry.remotePort,
			"",              // no pod name
			entry.container, // container name
			"",              // no namespace
			"",              // no port name
			"container",
			entry.resourceName,
			entry.resourceAddress,
		)

		eventV2.PortForwarded(
			entry.localPort,
			entry.remotePort,
			"",              // no pod name
			entry.container, // container name
			"",              // no namespace
			"",              // no port name
			"container",
			entry.resourceName,
			entry.resourceAddress,
		)

		output.Green.Fprintln(out,
			fmt.Sprintf("[%s] Forwarding container port %s -> local port %s:%d",
				entry.container,
				entry.remotePort.String(),
				entry.resourceAddress,
				entry.localPort,
			))
	}
}
