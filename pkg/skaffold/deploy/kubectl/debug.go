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
	"strings"

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

		if transformManifest(m, builds) && logrus.IsLevelEnabled(logrus.DebugLevel) {
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
func transformManifest(m map[interface{}]interface{}, builds []build.Artifact) bool {
	switch {
	case m["kind"] == "Pod" && m["apiVersion"] == "v1":
		return transformPodSpec(m, builds)
	case m["kind"] == "Deployment" && (m["apiVersion"] == "extensions/v1beta1" || m["apiVersion"] == "apps/v1"):
		return transformDeployment(m, builds)
	default:
		logrus.Debugf("skipping manifest kind:%v apiVersion:%v\n", m["kind"], m["apiVersion"])
		return false
	}
}

// transformDeployment attempts to configure a deployment's podspec for debugging.
// Returns true if changed, false otherwise.
func transformDeployment(m map[interface{}]interface{}, builds []build.Artifact) bool {
	template, ok := traverse(m, "spec", "template")
	if ok {
		podSpec, ok := template.(map[interface{}]interface{})
		if ok {
			return transformPodSpec(podSpec, builds)
		}
	}
	return false
}

const (
	// JVM indicates an application that requires the Java Virtual Machine
	JVM = "jvm"
	// UNKNOWN indicates that runtime cannot be determined
	UNKNOWN = ""
)

// transformPodSpec attempts to configure a podspec for debugging.
// Returns true if changed, false otherwise.
func transformPodSpec(podSpec map[interface{}]interface{}, builds []build.Artifact) bool {
	containers, found := traverse(podSpec, "spec", "containers")
	if !found {
		return false
	}
	switch containers := containers.(type) {
	case []interface{}: // can't use []map[interface{}]interface{} !?
		// configurations maps a container-name -> debugging configuration description
		configurations := make(map[string]map[string]interface{})
		for _, container := range containers {
			switch container := container.(type) {
			case map[interface{}]interface{}:
				containerName := container["name"].(string) // containers are required have unique name
				image := container["image"].(string)
				// we only reconfigure build artifacts
				if artifact := findArtifact(image, builds); artifact != nil {
					logrus.Debugf("Found artifact for image %v", image)
					if configuration := transformContainer(container, *artifact); configuration != nil {
						configurations[containerName] = configuration
					}
					// fixme: add this artifact to the watch list?
				} else {
					logrus.Debugf("Ignoring image %v for debugging: no corresponding build artifact", image)
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

// findArtifact finds the corresponding artifact for the given image
func findArtifact(image string, builds []build.Artifact) *build.Artifact {
	for _, artifact := range builds {
		if image == artifact.ImageName || image == artifact.Tag {
			return &artifact
		}
	}
	return nil
}

// imageConfiguration captures information from a docker/oci image configuration
type imageConfiguration struct {
	labels     map[string]string
	env        map[string]string
	entrypoint []string
	arguments  []string
}

func retrieveImageConfiguration(image string, artifact build.Artifact) imageConfiguration {
	// TODO: obtain the built image configuration; note that the build artifact's
	// name not actually correspond to the image name (for example, when built to the local docker daemon)
	return imageConfiguration{env: map[string]string{"JAVA_VERSION": "8"}}
}

func guessRuntime(config imageConfiguration) string {
	if _, found := config.env["JAVA_TOOL_OPTIONS"]; found {
		return JVM
	}
	if _, found := config.env["JAVA_VERSION"]; found {
		return JVM
	}
	return UNKNOWN
}

// transformContainer rewrites the container definition to enable debugging and returns a debugging configuration description
func transformContainer(container map[interface{}]interface{}, artifact build.Artifact) map[string]interface{} {
	containerName := container["name"].(string)
	image := container["image"].(string)
	config := retrieveImageConfiguration(image, artifact)

	// update image configuration values with those set in the k8s manifest
	switch env := container["env"].(type) {
	case map[interface{}]interface{}:
		for key, value := range env {
			config.env[key.(string)] = value.(string)
		}
	}
	if cmd, found := container["command"]; found {
		config.entrypoint = cmd.([]string)
	}
	if args, found := container["args"]; found {
		config.arguments = args.([]string)
	}

	switch guessRuntime(config) {
	case JVM:
		logrus.Debugf("Configuring %v for JVM", containerName)
		return configureJvmDebugging(container, config)
	default:
		logrus.Debugf("Unable to determine runtime for %v\n", containerName)
		return nil
	}
}

// configureJvmDebugging configured a container definition for JVM debugging.
// Returns a simple map describing the debug configuration details.
func configureJvmDebugging(container map[interface{}]interface{}, config imageConfiguration) map[string]interface{} {
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

// envAsMap turns an array of enviroment "NAME=value" strings into a map
func envAsMap(env []string) map[string]string {
	result := make(map[string]string)
	for _, pair := range env {
		s := strings.SplitN(pair, "=", 2)
		result[s[0]] = s[1]
	}
	return result
}
