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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
)

// portAllocator is a function that takes a desired port and returns an available port
// Ports are normally uint16 but Kubernetes ContainerPort.containerPort is an integer
type portAllocator func(int32) int32

// ApplyDebuggingTransforms applies language-platform-specific transforms to a list of manifests.
func ApplyDebuggingTransforms(l ManifestList, builds []build.Artifact) (ManifestList, error) {
	var updated ManifestList
	decode := scheme.Codecs.UniversalDeserializer().Decode

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	encode := func(o runtime.Object) ([]byte, error) {
		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		if err := s.Encode(o, w); err != nil {
			return nil, err
		}
		w.Flush()
		return b.Bytes(), nil
	}

	for _, manifest := range l {

		obj, _, err := decode(manifest, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if transformManifest(obj, builds) {
			manifest, err = encode(obj)
			if err != nil {
				return nil, errors.Wrap(err, "marshalling yaml")
			}
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				logrus.Debugln("Applied debugging transform:\n", string(manifest))
			}
		}
		updated = append(updated, manifest)
	}

	return updated, nil
}

// transformManifest attempts to configure a manifest for debugging.
// Returns true if changed, false otherwise.
func transformManifest(obj runtime.Object, builds []build.Artifact) bool {
	// FIXME: add other types
	switch o := obj.(type) {
	case *v1.Pod:
		return transformPodSpec(&o.ObjectMeta, &o.Spec, builds)
	case *appsv1.Deployment:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, builds)
	default:
		logrus.Debugf("skipping unknown object: %v\n", obj)
		return false
	}
}

const (
	// JVM indicates an application that requires the Java Virtual Machine
	JVM = "jvm"
	// NODEJS indicates an application that requires NodeJS
	NODEJS = "nodejs"
	// UNKNOWN indicates that runtime cannot be determined
	UNKNOWN = ""
)

// transformPodSpec attempts to configure a podspec for debugging.
// Returns true if changed, false otherwise.
func transformPodSpec(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, builds []build.Artifact) bool {
	configurations := make(map[string]map[string]interface{})
	portAlloc := func(desiredPort int32) int32 {
		return allocatePort(podSpec, desiredPort)
	}
	// containers are required have unique name within a pod
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		// we only reconfigure build artifacts
		if artifact := findArtifact(container.Image, builds); artifact != nil {
			logrus.Debugf("Found artifact for image [%s]", container.Image)
			if configuration, err := transformContainer(container, *artifact, portAlloc); err == nil {
				configurations[container.Name] = configuration
				// todo: add this artifact to the watch list?
			} else {
				logrus.Infof("Could not configure image [%s] for debugging: %v", container.Image, err)
			}
		} else {
			logrus.Debugf("Ignoring image [%s] for debugging: no corresponding build artifact", container.Image)
		}
	}
	if len(configurations) > 0 {
		if metadata.Annotations == nil {
			metadata.Annotations = make(map[string]string)
		}
		metadata.Annotations["debug.cloud.google.com/config"] = encodeConfigurations(configurations)
		return true
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

// allocatePort walkas the podSpec's containers looking for an available port that is as close to desiredPort as possible
// We deal with wrapping and avoid allocating ports < 1024
func allocatePort(podSpec *v1.PodSpec, desiredPort int32) int32 {
	var maxPort int32 = 65535 // ports are normally [1-65535]
	// Theoretically the port-space could be full, but that seems unlikely
	for {
		if desiredPort < 1024 || desiredPort > maxPort {
			desiredPort = 1024 // skip reserved ports
		}
		windowSize := maxPort - desiredPort + 1
		// check pod containers for the next 20 ports
		if windowSize > 20 {
			windowSize = 20
		}
		var allocated = make([]bool, windowSize)
		for _, container := range podSpec.Containers {
			for _, portSpec := range container.Ports {
				if portSpec.ContainerPort >= desiredPort && portSpec.ContainerPort-desiredPort < windowSize {
					allocated[portSpec.ContainerPort-desiredPort] = true
				}
			}
		}
		for i := range allocated {
			if !allocated[i] {
				return desiredPort + int32(i)
			}
		}
		// on to the next window
		desiredPort += windowSize
	}
	// NOTREACHED
}

// imageConfiguration captures information from a docker/oci image configuration
type imageConfiguration struct {
	labels     map[string]string
	env        map[string]string
	entrypoint []string
	arguments  []string
}

func retrieveImageConfiguration(image string, artifact build.Artifact) imageConfiguration {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := artifact.Config(ctx)
	if err != nil {
		logrus.Errorf("unable to retrieve image configuration for [%q]: %v", image, err)
		return imageConfiguration{}
	}
	return imageConfiguration{
		env:        envAsMap(config.Env),
		entrypoint: config.Entrypoint,
		arguments:  config.Cmd,
		labels:     config.Labels,
	}
}

// transformContainer rewrites the container definition to enable debugging. 
// Returns a debugging configuration description or an error if the rewrite was unsuccessful.
func transformContainer(container *v1.Container, artifact build.Artifact, portAlloc portAllocator) (map[string]interface{}, error) {
	config := retrieveImageConfiguration(container.Image, artifact)

	// update image configuration values with those set in the k8s manifest
	for _, envVar := range container.Env {
		// FIXME handle ValueFrom?
		config.env[envVar.Name] = envVar.Value
	}

	if len(container.Command) > 0 {
		config.entrypoint = container.Command
	}
	if len(container.Args) > 0 {
		config.arguments = container.Args
	}

	switch guessRuntime(config) {
	case JVM:
		logrus.Infof("Configuring [%s] for JVM debugging", container.Name)
		return configureJvmDebugging(container, config, portAlloc), nil
	case NODEJS:
		logrus.Infof("Configuring [%s] for node.js debugging", container.Name)
		return configureNodeJSDebugging(container, config, portAlloc), nil
	default:
		return nil, errors.Errorf("unable to determine runtime for [%s]", container.Name)
	}
}

func guessRuntime(config imageConfiguration) string {
	if _, found := config.env["JAVA_TOOL_OPTIONS"]; found {
		return JVM
	}
	if _, found := config.env["JAVA_VERSION"]; found {
		return JVM
	}
	if _, found := config.env["NODE_VERSION"]; found {
		return NODEJS
	}
	if len(config.entrypoint) > 0 {
		if config.entrypoint[0] == "java" || strings.HasSuffix(config.entrypoint[0], "/java") {
			return JVM
		}
		if config.entrypoint[0] == "node" || strings.HasSuffix(config.entrypoint[0], "/node") {
			return NODEJS
		}
	}
	if len(config.arguments) > 0 {
		if config.arguments[0] == "java" || strings.HasSuffix(config.arguments[0], "/java") {
			return JVM
		}
		if config.arguments[0] == "node" || strings.HasSuffix(config.arguments[0], "/node") {
			return NODEJS
		}
	}
	return UNKNOWN
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
