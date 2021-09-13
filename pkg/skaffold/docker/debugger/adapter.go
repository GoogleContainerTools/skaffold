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

package debugger

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

type DockerAdapter struct {
	cfg        *container.Config
	executable *types.ExecutableContainer
}

func NewAdapter(cfg *container.Config) *DockerAdapter {
	return &DockerAdapter{
		cfg:        cfg,
		executable: ExecutableContainerForConfig(cfg),
	}
}

func (d *DockerAdapter) GetContainer() *types.ExecutableContainer {
	return d.executable
}

func ExecutableContainerForConfig(cfg *container.Config) *types.ExecutableContainer {
	return &types.ExecutableContainer{
		Name:    cfg.Image,
		Command: cfg.Cmd,
		Env:     dockerEnvToContainerEnv(cfg.Env),
		Ports:   dockerPortsToContainerPorts(cfg.ExposedPorts),
	}
}

// Apply transfers the configuration changes from the intermediate container
// to the underlying container config.
// Since container.Config doesn't have an Args field, we combine the ExecutableContainer
// Args with the Cmd and slot them in there.
func (d *DockerAdapter) Apply() {
	d.cfg.Cmd = d.executable.Command
	d.cfg.Cmd = append(d.cfg.Cmd, d.executable.Args...)
	d.cfg.Env = containerEnvToDockerEnv(d.executable.Env)
	d.cfg.ExposedPorts = containerPortsToDockerPorts(d.executable.Ports)
}

func dockerEnvToContainerEnv(dockerEnv []string) types.ContainerEnv {
	env := make(map[string]string, len(dockerEnv))
	var order []string
	for _, entry := range dockerEnv {
		parts := strings.SplitN(entry, "=", 2) // split to max 2 substrings, `=` is a valid character in the env value
		if len(parts) != 2 {
			log.Entry(context.TODO()).Warnf("malformed env entry %s: skipping", entry)
			continue
		}
		order = append(order, parts[0])
		env[parts[0]] = parts[1]
	}
	return types.ContainerEnv{
		Order: order,
		Env:   env,
	}
}

func containerEnvToDockerEnv(env types.ContainerEnv) []string {
	var dockerEnv []string
	for _, k := range env.Order {
		dockerEnv = append(dockerEnv, fmt.Sprintf("%s=%s", k, env.Env[k]))
	}
	return dockerEnv
}

func dockerPortsToContainerPorts(ports nat.PortSet) []types.ContainerPort {
	var containerPorts []types.ContainerPort
	for k := range ports {
		// net.Port is a typecast of a string
		containerPorts = append(containerPorts, types.ContainerPort{
			Name:          string(k),
			ContainerPort: int32(k.Int()),
			Protocol:      k.Proto(),
		})
	}
	return containerPorts
}

func containerPortsToDockerPorts(containerPorts []types.ContainerPort) nat.PortSet {
	dockerPorts := make(nat.PortSet, len(containerPorts))
	for _, port := range containerPorts {
		portStr := strconv.Itoa(int(port.ContainerPort))
		dockerPort, err := nat.NewPort(port.Protocol, portStr)
		if err != nil {
			log.Entry(context.TODO()).Warnf("error translating port %s - debug might not work correctly!", portStr)
		}
		dockerPorts[dockerPort] = struct{}{}
	}
	return dockerPorts
}
