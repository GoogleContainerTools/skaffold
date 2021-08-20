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

package adapter

import (
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
)

type Adapter struct {
	container  *v1.Container
	executable *types.ExecutableContainer
	valueFrom  map[string]*v1.EnvVarSource
}

func NewAdapter(c *v1.Container) *Adapter {
	return &Adapter{
		container:  c,
		executable: ExecutableContainerFromK8sContainer(c),
		valueFrom:  holdValueFrom(c.Env),
	}
}

func (a *Adapter) GetContainer() *types.ExecutableContainer {
	return a.executable
}

// Apply takes the relevant fields from the operable container
// and applies them to the referenced v1.Container from the manifest's pod spec
func (a *Adapter) Apply() {
	a.container.Args = a.executable.Args
	a.container.Command = a.executable.Command
	a.container.Env = containerEnvToK8sEnv(a.executable.Env, a.valueFrom)
	a.container.Ports = containerPortsToK8sPorts(a.executable.Ports)
}

// ExecutableContainerFromK8sContainer creates an instance of an operableContainer
// from a v1.Container reference. This object will be passed around to accept
// transforms, and will eventually overwrite fields from the creating v1.Container
// in the manifest-under-transformation's pod spec.
func ExecutableContainerFromK8sContainer(c *v1.Container) *types.ExecutableContainer {
	return &types.ExecutableContainer{
		Command: c.Command,
		Args:    c.Args,
		Env:     k8sEnvToContainerEnv(c.Env),
		Ports:   k8sPortsToContainerPorts(c.Ports),
	}
}

func k8sEnvToContainerEnv(k8sEnv []v1.EnvVar) types.ContainerEnv {
	env := make(map[string]string, len(k8sEnv))
	var order []string
	for _, entry := range k8sEnv {
		order = append(order, entry.Name)
		env[entry.Name] = entry.Value
	}
	return types.ContainerEnv{
		Order: order,
		Env:   env,
	}
}

// ValueFrom isn't handled by the debug code when altering the env vars.
// holdValueFrom stores all ValueFrom values as they are on the adapter,
// which will then be put back in place by Apply() later.
func holdValueFrom(env []v1.EnvVar) map[string]*v1.EnvVarSource {
	from := make(map[string]*v1.EnvVarSource, len(env))
	for _, entry := range env {
		from[entry.Name] = entry.ValueFrom
	}
	return from
}

func containerEnvToK8sEnv(env types.ContainerEnv, valueFrom map[string]*v1.EnvVarSource) []v1.EnvVar {
	var k8sEnv []v1.EnvVar
	for _, k := range env.Order {
		k8sEnv = append(k8sEnv, v1.EnvVar{
			Name:      k,
			Value:     env.Env[k],
			ValueFrom: valueFrom[k],
		})
	}
	return k8sEnv
}

func k8sPortsToContainerPorts(k8sPorts []v1.ContainerPort) []types.ContainerPort {
	var containerPorts []types.ContainerPort
	for _, port := range k8sPorts {
		containerPorts = append(containerPorts, types.ContainerPort{
			Name:          port.Name,
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
			Protocol:      string(port.Protocol),
			HostIP:        port.HostIP,
		})
	}
	return containerPorts
}

func containerPortsToK8sPorts(containerPorts []types.ContainerPort) []v1.ContainerPort {
	var k8sPorts []v1.ContainerPort
	for _, port := range containerPorts {
		k8sPorts = append(k8sPorts, v1.ContainerPort{
			Name:          port.Name,
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
			Protocol:      v1.Protocol(port.Protocol),
			HostIP:        port.HostIP,
		})
	}
	return k8sPorts
}
