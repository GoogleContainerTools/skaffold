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

package dockerport

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
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

/*
	allocatePorts converts PortForwardResources into docker.PortSet objects, and combines them with
	pre-configured debug bindings into one docker.PortMap. The debug bindings will have their
	requested host ports validated against the port tracker, and modified if a port collision is found.

	These ports are added to the provided container configuration's port set, and the bindings
	are returned to be passed to ContainerCreate on Deploy to expose container ports on the host.
*/

func (pm *PortManager) AllocatePorts(containerName string, pf []*latest.PortForwardResource, cfg *container.Config, debugBindings nat.PortMap) (nat.PortMap, error) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	m := make(nat.PortMap)
	var entries []containerPortForwardEntry
	if cfg.ExposedPorts == nil {
		cfg.ExposedPorts = nat.PortSet{}
	}
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

	// we can't modify the existing debug bindings in place, since they are not passed by reference.
	// instead, copy each binding and modify the copy, then insert into a new map and return that.
	for port, bindings := range debugBindings {
		modifiedBindings := make([]nat.PortBinding, len(bindings))
		for i, b := range bindings {
			newBinding := nat.PortBinding{HostIP: b.HostIP, HostPort: b.HostPort}
			hostPort, err := strconv.Atoi(newBinding.HostPort)
			if err != nil {
				return nil, err
			}
			localPort := GetAvailablePort(newBinding.HostIP, hostPort, &pm.portSet)
			if localPort != hostPort {
				newBinding.HostPort = strconv.Itoa(localPort)
			}
			ports = append(ports, localPort)
			cfg.ExposedPorts[port] = struct{}{}
			entries = append(entries, containerPortForwardEntry{
				container:       containerName,
				resourceAddress: newBinding.HostIP,
				localPort:       int32(localPort),
				remotePort: schemautil.IntOrString{
					Type:   schemautil.Int,
					IntVal: port.Int(),
				},
			})
			modifiedBindings[i] = newBinding
		}
		m[port] = modifiedBindings
	}
	pm.containerPorts[containerName] = ports

	// register entries on the port manager, to be issued by the event handler later
	pm.entries = append(pm.entries, entries...)
	return m, nil
}

func (pm *PortManager) RelinquishPorts(containerName string) {
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
