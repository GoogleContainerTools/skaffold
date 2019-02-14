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

// configureJvmDebugging configured a container definition for JVM debugging.
// Returns a simple map describing the debug configuration details.
func configureJvmDebugging(container map[interface{}]interface{}, config imageConfiguration) map[string]interface{} {
	env, ok := container["env"].([]interface{}) // []map[interface{}]interface{}
	if !ok {
		env = make([]interface{},0) ///[]map[interface{}]interface{}
	}
	// FIXME try to find existing JAVA_TOOL_OPTIONS or jdwp command argument
	javaToolOptions := map[interface{}]interface{}{
		"name":  "JAVA_TOOL_OPTIONS",
		"value": "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y",
	}
	container["env"] = append(env, javaToolOptions)

	ports, ok := container["ports"].([]interface{}) // []map[string]interface{}
	if !ok {
		ports = make([]interface{},0) ///[]map[interface{}]interface{}
	}
	jdwpPort := map[interface{}]interface{}{
		"name": "jdwp",
		"containerPort": 5005,
	}
	container["ports"] = append(ports, jdwpPort)

	return map[string]interface{}{
		"runtime": "jvm",
		"jdwp":    5005,
	}
}
