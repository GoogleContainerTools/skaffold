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

package kubectl

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestConfigureJvmDebugging(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration imageConfiguration
		result        v1.Container
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{v1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{v1.ContainerPort{Name: "jdwp", ContainerPort: 5005}},
			},
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{v1.ContainerPort{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: imageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{v1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{v1.ContainerPort{Name: "http-server", ContainerPort: 8080}, v1.ContainerPort{Name: "jdwp", ContainerPort: 5005}},
			},
		},
	}
	var identity portAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			configureJvmDebugging(&test.containerSpec, test.configuration, identity)
			testutil.CheckDeepEqual(t, test.result, test.containerSpec)
		})
	}
}
