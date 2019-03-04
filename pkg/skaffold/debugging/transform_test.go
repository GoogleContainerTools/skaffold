/*
Copyright 2018 The Skaffold Authors

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

package debugging

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
)

func TestAllocatePort(t *testing.T) {
	// helper function to create a container
	containerWithPorts := func(ports ...int32) v1.Container {
		var created []v1.ContainerPort
		for _, port := range ports {
			created = append(created, v1.ContainerPort{ContainerPort: port})
		}
		return v1.Container{Ports: created}
	}

	tests := []struct {
		description string
		pod         v1.PodSpec
		desiredPort int32
		result      int32
	}{
		{
			description: "simple",
			pod:         v1.PodSpec{},
			desiredPort: 5005,
			result:      5005,
		},
		{
			description: "finds next available port",
			pod: v1.PodSpec{Containers: []v1.Container{
				containerWithPorts(5005, 5007),
				containerWithPorts(5008, 5006),
			}},
			desiredPort: 5005,
			result:      5009,
		},
		{
			description: "skips reserved",
			pod:         v1.PodSpec{},
			desiredPort: 1,
			result:      1024,
		},
		{
			description: "skips 0",
			pod:         v1.PodSpec{},
			desiredPort: 0,
			result:      1024,
		},
		{
			description: "wraps too large",
			pod:         v1.PodSpec{},
			desiredPort: 65536,
			result:      1024,
		},
		{
			description: "skips negative",
			pod:         v1.PodSpec{},
			desiredPort: -1,
			result:      1024,
		},
		{
			description: "wraps at 65535",
			pod: v1.PodSpec{Containers: []v1.Container{
				containerWithPorts(65535),
			}},
			desiredPort: 65535,
			result:      1024,
		},
		{
			description: "wraps and skips",
			pod: v1.PodSpec{Containers: []v1.Container{
				containerWithPorts(1025, 65535),
				containerWithPorts(65534, 1024),
			}},
			desiredPort: 65534,
			result:      1026,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := allocatePort(&test.pod, test.desiredPort)
			testutil.CheckDeepEqual(t, test.result, result)
		})
	}
}
