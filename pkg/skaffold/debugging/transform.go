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

package debugging

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// portAllocator is a function that takes a desired port and returns an available port
// Ports are normally uint16 but Kubernetes ContainerPort.containerPort is an integer
type portAllocator func(int32) int32

// configurationRetriever retrieves an container image configuration
type configurationRetriever func(string) (imageConfiguration, error)

// imageConfiguration captures information from a docker/oci image configuration
type imageConfiguration struct {
	labels     map[string]string
	env        map[string]string
	entrypoint []string
	arguments  []string
}

// containerTransformer transforms a container definition
type containerTransformer interface {
	// IsApplicable determines if this container is suitable to be transformed.
	IsApplicable(config imageConfiguration) bool

	// Apply configures a container definition for debugging, returning a simple map describing the debug configuration details or `nil` if it could not be done
	Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) map[string]interface{}
}

var containerTransforms []containerTransformer

// transformManifest attempts to configure a manifest for debugging.
// Returns true if changed, false otherwise.
func transformManifest(obj runtime.Object, retrieveImageConfiguration configurationRetriever) bool {
	switch o := obj.(type) {
	case *v1.Pod:
		return transformPodSpec(&o.ObjectMeta, &o.Spec, retrieveImageConfiguration)
	case *v1.ReplicationController:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.Deployment:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.DaemonSet:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.ReplicaSet:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.StatefulSet:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *batchv1.Job:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)

	default:
		logrus.Debugf("skipping unknown object: %T (%v)\n", obj.GetObjectKind(), obj)
		return false
	}
}

// transformPodSpec attempts to configure a podspec for debugging.
// Returns true if changed, false otherwise.
func transformPodSpec(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, retrieveImageConfiguration configurationRetriever) bool {
	portAlloc := func(desiredPort int32) int32 {
		return allocatePort(podSpec, desiredPort)
	}
	// containers are required to have unique name within a pod
	configurations := make(map[string]map[string]interface{})
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		// we only reconfigure build artifacts
		if configuration, err := transformContainer(container, retrieveImageConfiguration, portAlloc); err == nil {
			configurations[container.Name] = configuration
			// todo: add this artifact to the watch list?
		} else {
			logrus.Infof("Image [%s] not configured for debugging: %v", container.Image, err)
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

// transformContainer rewrites the container definition to enable debugging.
// Returns a debugging configuration description or an error if the rewrite was unsuccessful.
func transformContainer(container *v1.Container, retrieveImageConfiguration configurationRetriever, portAlloc portAllocator) (map[string]interface{}, error) {
	var config imageConfiguration
	config, err := retrieveImageConfiguration(container.Image)
	if err != nil {
		return nil, err
	}

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

	for _, transform := range containerTransforms {
		if transform.IsApplicable(config) {
			return transform.Apply(container, config, portAlloc), nil
		}
	}
	return nil, errors.Errorf("unable to determine runtime for [%s]", container.Name)
}

func encodeConfigurations(configurations map[string]map[string]interface{}) string {
	bytes, err := json.Marshal(configurations)
	if err != nil {
		return ""
	}
	return string(bytes)
}
