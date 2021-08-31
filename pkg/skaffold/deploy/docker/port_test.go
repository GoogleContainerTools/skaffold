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
	"testing"

	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types/container"
)

func TestGetPorts(t *testing.T) {
	tests := []struct {
		name      string
		resources map[int]*v1.PortForwardResource // we map local port to resources for ease of testing
	}{
		{
			name: "one port, one resource",
			resources: map[int]*v1.PortForwardResource{
				9000: {
					Port: util.IntOrString{
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
			resources: map[int]*v1.PortForwardResource{
				1234: {
					Port: util.IntOrString{
						IntVal: 20,
						StrVal: "20",
					},
					Address:   "192.168.999.999",
					LocalPort: 1234,
				},
				4321: {
					Port: util.IntOrString{
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
			pm := NewPortManager()
			cfg := container.Config{}
			m, err := pm.getPorts(test.name, collectResources(test.resources), &cfg)
			for port := range cfg.ExposedPorts { // the image config's PortSet contains the local ports, so we grab the bindings keyed off these
				bindings := m[port]
				t.CheckDeepEqual(len(bindings), 1) // we always have a 1-1 mapping of resource to binding
				t.CheckError(false, err)           // shouldn't error, unless GetAvailablePort is broken
				t.CheckDeepEqual(bindings[0].HostIP, test.resources[port.Int()].Address)
				t.CheckDeepEqual(bindings[0].HostPort, test.resources[port.Int()].Port.StrVal)
			}
		})
	}
}

func collectResources(resourceMap map[int]*v1.PortForwardResource) []*v1.PortForwardResource {
	var resources []*v1.PortForwardResource
	for _, r := range resourceMap {
		resources = append(resources, r)
	}
	return resources
}
