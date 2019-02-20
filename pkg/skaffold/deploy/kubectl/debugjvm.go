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

package kubectl

import (
"fmt"
	v1 "k8s.io/api/core/v1"
)

// configureJvmDebugging configured a container definition for JVM debugging.
// Returns a simple map describing the debug configuration details.
func configureJvmDebugging(container *v1.Container, config imageConfiguration, portAlloc portAllocator) map[string]interface{} {
	// no standard port for JDWP; most examples use 5005 or 8000
	port := portAlloc(5005)

	// FIXME try to find existing JAVA_TOOL_OPTIONS or jdwp command argument
	javaToolOptions := v1.EnvVar{
		Name:  "JAVA_TOOL_OPTIONS",
		Value: fmt.Sprintf("-agentlib:jdwp=transport=dt_socket,server=y,address=%d,suspend=n,quiet=y", port),
	}
	container.Env = append(container.Env, javaToolOptions)

	jdwpPort := v1.ContainerPort{
		Name:          "jdwp",
		ContainerPort: port,
	}
	container.Ports = append(container.Ports, jdwpPort)

	return map[string]interface{}{
		"runtime": "jvm",
		"jdwp":    port,
	}
}
