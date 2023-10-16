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
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/container"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestAllocatePorts(t *testing.T) {
	tests := []struct {
		name      string
		resources map[int]*latest.PortForwardResource // we map local port to resources for ease of testing
	}{
		{
			name: "one port, one resource",
			resources: map[int]*latest.PortForwardResource{
				9000: {
					Type: "container",
					Port: schemautil.IntOrString{
						IntVal: 9000,
						StrVal: "9000",
					},
					Address:   "127.0.0.1",
					LocalPort: 9000,
				},
			},
		},
		{
			name: "two ports, two resources",
			resources: map[int]*latest.PortForwardResource{
				1234: {
					Type: "container",
					Port: schemautil.IntOrString{
						IntVal: 20,
						StrVal: "20",
					},
					Address:   "192.168.999.999",
					LocalPort: 1234,
				},
				4321: {
					Type: "container",
					Port: schemautil.IntOrString{
						IntVal: 8080,
						StrVal: "8080",
					},
					Address:   "localhost",
					LocalPort: 4321,
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&GetAvailablePort, func(_ string, port int, _ *util.PortSet) int {
				return port
			})
			pm := NewPortManager()
			cfg := container.Config{}
			m, err := pm.AllocatePorts(test.name, collectResources(test.resources), &cfg, nil)
			for containerPort := range cfg.ExposedPorts { // the image config's PortSet contains the local ports, so we grab the bindings keyed off these
				bindings := m[containerPort]
				t.CheckDeepEqual(len(bindings), 1) // we always have a 1-1 mapping of resource to binding
				t.CheckError(false, err)           // shouldn't error, unless GetAvailablePort is broken
				resource := resourceFromContainerPort(test.resources, containerPort.Int())
				t.CheckDeepEqual(bindings[0].HostIP, resource.Address)
				t.CheckDeepEqual(bindings[0].HostPort, strconv.Itoa(resource.LocalPort))
			}
		})
	}
}

func collectResources(resourceMap map[int]*latest.PortForwardResource) []*latest.PortForwardResource {
	var resources []*latest.PortForwardResource
	for _, r := range resourceMap {
		resources = append(resources, r)
	}
	return resources
}

func resourceFromContainerPort(resourceMap map[int]*latest.PortForwardResource, containerPort int) *latest.PortForwardResource {
	for _, resource := range resourceMap {
		if resource.Port.IntVal == containerPort {
			return resource
		}
	}
	return nil
}
