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
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
)

// ApplyDebuggingTransforms applies language-platform-specific transforms to a list of manifests.
func ApplyDebuggingTransforms(l ManifestList, builds []build.Artifact) (ManifestList, error) {
	var updated ManifestList
	for _, manifest := range l {
		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(manifest, m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if len(m) == 0 {
			continue
		}

		if transformManifest(m) && logrus.IsLevelEnabled(logrus.DebugLevel) {
			bytes, _ := yaml.Marshal(m)
			logrus.Debugln("Applied debugging transforms:\n", string(bytes))
		}

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling yaml")
		}
		updated = append(updated, updatedManifest)
	}

	return updated, nil
}

// transformManifest attempts to configure a manifest for debugging.
// Returns true if changed, false otherwise.
func transformManifest(m map[interface{}]interface{}) bool {
	switch {
	case m["kind"] == "Pod" && m["apiVersion"] == "v1":
		return transformPodSpec(m)
	case m["kind"] == "Deployment" && m["apiVersion"] == "extensions/v1beta1":
		return transformDeployment(m)
	default:
		return false
	}
}

// transformDeployment attempts to configure a deployment's podspec for debugging.
// Returns true if changed, false otherwise.
func transformDeployment(m map[interface{}]interface{}) bool {
	template, ok := traverse(m, "spec", "template")
	if ok {
		podSpec, ok := template.(map[interface{}]interface{})
		if ok {
			return transformPodSpec(podSpec)
		}
	}
	return false
}

// transformPodSpec attempts to configure a podspec for debugging.
// Returns true if changed, false otherwise.
func transformPodSpec(podSpec map[interface{}]interface{}) bool {
	containers, ok := traverse(podSpec, "spec", "containers")
	if !ok {
		return false
	}
	switch containers := containers.(type) {
	case []interface{}: // can't use []map[interface{}]interface{} !?
		configurations := make(map[string]map[string]interface{})
		for _, container := range containers {
			switch container := container.(type) {
			case map[interface{}]interface{}:
				containerName := container["name"].(string) // containers should have unique name
				// FIXME determine language technology and configure
				if jvmConfig := configureJvmDebugging(container); jvmConfig != nil {
					configurations[containerName] = jvmConfig
				}
			}
		}
		if len(configurations) > 0 {
			annotations := traverseToMap(podSpec, "metadata", "annotations")
			annotations["debug.cloud.google.com/config"] = encodeConfigurations(configurations)
			return true
		}
	}
	return false
}

// configureJvmDebugging configured a container definition for JVM debugging.
// Returns a simple map describing the debug configuration details.
func configureJvmDebugging(container map[interface{}]interface{}) map[string]interface{} {
	env, ok := container["env"].([]map[interface{}]interface{})
	if !ok {
		env = make([]map[interface{}]interface{}, 0)
	}
	// FIXME try to find existing JAVA_TOOL_OPTIONS or jdwp command argument
	javaToolOptions := map[interface{}]interface{}{
		"name":  "JAVA_TOOL_OPTIONS",
		"value": "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y",
	}
	container["env"] = append(env, javaToolOptions)

	ports, ok := container["ports"].([]map[interface{}]interface{})
	if !ok {
		ports = make([]map[interface{}]interface{}, 0)
	}
	containerPort := map[interface{}]interface{}{
		"containerPort": 5005,
	}
	container["ports"] = append(ports, containerPort)

	return map[string]interface{}{
		"runtime": "jvm",
		"jdwp":    5005,
	}
}

func encodeConfigurations(configurations map[string]map[string]interface{}) string {
	bytes, err := json.Marshal(configurations)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func traverse(m map[interface{}]interface{}, keys ...string) (value interface{}, found bool) {
	for index, k := range keys {
		value, found = m[k]
		if !found {
			return
		}
		if index == len(keys)-1 {
			return
		}
		switch t := value.(type) {
		case map[interface{}]interface{}:
			m = t
		default:
			break
		}
	}
	return nil, false
}

// traverse the path, ensuring each point is a map
func traverseToMap(m map[interface{}]interface{}, keys ...string) map[interface{}]interface{} {
	var result map[interface{}]interface{}
	for _, k := range keys {
		value := m[k]
		switch t := value.(type) {
		case map[interface{}]interface{}:
			result = t
		default:
			result = make(map[interface{}]interface{})
			m[k] = result
		}
		m = result
	}
	return result
}
