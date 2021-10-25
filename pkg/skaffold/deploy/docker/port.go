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
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var GetAvailablePort = util.GetAvailablePort // For testing

type containerPortForwardEntry struct {
	container       string
	resourceName    string
	resourceAddress string
	localPort       int32
	remotePort      schemautil.IntOrString
}

type PortManager struct {
	containerPorts map[string][]int // maps containers to the ports they have allocated
	portSet        util.PortSet
	entries        []containerPortForwardEntry // reference shared with DockerForwarder so output is issued in the correct phase of the dev loop
	lock           sync.Mutex
}

func NewPortManager() *PortManager {
	return &PortManager{
		containerPorts: make(map[string][]int),
		portSet:        util.PortSet{},
	}
}

func (pm *PortManager) Start(_ context.Context, out io.Writer) error {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.containerPortForwardEvents(out)
	return nil
}

func (pm *PortManager) Stop() {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.entries = nil
}

// getPorts converts PortForwardResources into docker.PortSet/PortMap objects.
// These ports are added to the provided container configuration's port set, and the bindings
// are returned to be passed to ContainerCreate on Deploy to expose container ports on the host.
// It also returns a list of containerPortForwardEntry, to be passed to the event handler
func (pm *PortManager) getPorts(containerName string, pf []*v2.PortForwardResource, cfg *container.Config) (nat.PortMap, error) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	m := make(nat.PortMap)
	var entries []containerPortForwardEntry
	var ports []int
	for _, p := range pf {
		if strings.ToLower(string(p.Type)) != "container" {
			log.Entry(context.TODO()).Debugf("skipping non-container port forward resource in Docker deploy: %s\n", p.Name)
			continue
		}
		localPort := GetAvailablePort(p.Address, p.LocalPort, &pm.portSet)
		ports = append(ports, localPort)
		port, err := nat.NewPort("tcp", p.Port.String())
		if err != nil {
			return nil, err
		}
		if cfg.ExposedPorts == nil {
			cfg.ExposedPorts = nat.PortSet{}
		}
		cfg.ExposedPorts[port] = struct{}{}
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
	pm.entries = append(pm.entries, entries...)
	return m, nil
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

func (pm *PortManager) containerPortForwardEvents(out io.Writer) {
	for _, entry := range pm.entries {
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
			fmt.Sprintf("[%s] Forwarding container port %s -> local port http://%s:%d",
				entry.container,
				entry.remotePort.String(),
				entry.resourceAddress,
				entry.localPort,
			))
	}
}
