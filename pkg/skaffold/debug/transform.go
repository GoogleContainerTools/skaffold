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

/*
The `debug` package transforms Kubernetes pod-bearing resources so as to configure containers
for remote debugging as suited for a container's runtime technology.  This package defines
a _container transformer_ interface. Each transformer implementation should do the following:

1. The transformer should modify the container's entrypoint, command arguments, and environment to enable debugging for the appropriate language runtime.
2. The transformer should expose the port(s) required to connect remote debuggers.
3. The transformer should identify any additional support files required to enable debugging (e.g., the `ptvsd` debugger for Python).
4. The transform should return metadata to describe the remote connection information.

Certain language runtimes require additional support files to enable remote debugging.
These support files are provided through a set of support images defined at `gcr.io/gcp-dev-tools/duct-tape/`
and defined at https://github.com/GoogleContainerTools/container-debug-support.
The appropriate image ID is returned by the language transformer.  These support images
are configured as initContainers on the pod and are expected to copy the debugging support
files into a support volume mounted at `/dbg`.  The expected convention is that each runtime's
files are placed in `/dbg/<runtimeId>`.  This same volume is then mounted into the
actual containers at `/dbg`.

As Kubernetes container objects don't actually carry metadata, we place this metadata on
the container's parent as an _annotation_; as a pod/podspec can have multiple containers, each of which may
be debuggable, we record this metadata using as a JSON object keyed by the container name.
Kubernetes requires that containers within a podspec are uniquely named.
For example, a pod with two containers named `microservice` and `adapter` may be:

  debug.cloud.google.com/config: '{
    "microservice":{"devtools":9229,"runtime":"nodejs"},
    "adapter":{"jdwp":5005,"runtime":"jvm"}
  }'

Each configuration is itself a JSON object with a `runtime` field identifying the
language runtime, and a set of runtime-specific fields describing connection information.
*/
package debug

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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

	// RuntimeSupportImage returns the associated duct-tape helper image required or empty string
	RuntimeSupportImage() string

	// Apply configures a container definition for debugging, returning a simple map describing the debug configuration details or `nil` if it could not be done
	Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) map[string]interface{}
}

// debuggingSupportVolume is the name of the volume used to hold language runtime debugging support files
const debuggingSupportFilesVolume = "debugging-support-files"

var containerTransforms []containerTransformer

// transformManifest attempts to configure a manifest for debugging.
// Returns true if changed, false otherwise.
func transformManifest(obj runtime.Object, retrieveImageConfiguration configurationRetriever) bool {
	one := int32(1)
	switch o := obj.(type) {
	case *v1.Pod:
		return transformPodSpec(&o.ObjectMeta, &o.Spec, retrieveImageConfiguration)
	case *v1.PodList:
		changed := false
		for i := range o.Items {
			if transformPodSpec(&o.Items[i].ObjectMeta, &o.Items[i].Spec, retrieveImageConfiguration) {
				changed = true
			}
		}
		return changed
	case *v1.ReplicationController:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.Deployment:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.DaemonSet:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.ReplicaSet:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *appsv1.StatefulSet:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)
	case *batchv1.Job:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration)

	default:
		group, version, _, description := describe(obj)
		if group == "apps" || group == "batch" {
			if version != "v1" {
				// treat deprecated objects as errors
				logrus.Errorf("deprecated versions not supported by debug: %s (%s)", description, version)
			} else {
				logrus.Warnf("no debug transformation for: %s", description)
			}
		} else {
			logrus.Debugf("no debug transformation for: %s", description)
		}
		return false
	}
}

// transformPodSpec attempts to configure a podspec for debugging.
// Returns true if changed, false otherwise.
func transformPodSpec(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, retrieveImageConfiguration configurationRetriever) bool {
	portAlloc := func(desiredPort int32) int32 {
		return allocatePort(podSpec, desiredPort)
	}
	// map of containers -> debugging configuration maps; k8s ensures that a pod's containers are uniquely named
	configurations := make(map[string]map[string]interface{})
	// the container images that require debugging support files
	var containersRequiringSupport []*v1.Container
	// the set of image IDs required to provide debugging support files
	requiredSupportImages := make(map[string]bool)
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		// the usual retriever returns an error for non-build artifacts
		imageConfig, err := retrieveImageConfiguration(container.Image)
		if err != nil {
			continue
		}
		// requiredImage, if not empty, is the image ID providing the debugging support files
		if configuration, requiredImage, err := transformContainer(container, imageConfig, portAlloc); err == nil {
			configurations[container.Name] = configuration
			if len(requiredImage) > 0 {
				logrus.Infof("%q requires debugging support image %q", container.Name, requiredImage)
				containersRequiringSupport = append(containersRequiringSupport, container)
				requiredSupportImages[requiredImage] = true
			}
			// todo: add this artifact to the watch list?
		} else {
			logrus.Infof("Image %q not configured for debugging: %v", container.Name, err)
		}
	}

	// check if we have any images requiring additional debugging support files
	if len(containersRequiringSupport) > 0 {
		logrus.Infof("Configuring installation of debugging support files")
		// we create the volume that will hold the debugging support files
		supportVolume := v1.Volume{Name: debuggingSupportFilesVolume, VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}}
		podSpec.Volumes = append(podSpec.Volumes, supportVolume)

		// this volume is mounted in the containers at `/dbg`
		supportVolumeMount := v1.VolumeMount{Name: debuggingSupportFilesVolume, MountPath: "/dbg"}
		// the initContainers are responsible for populating the contents of `/dbg`
		// TODO make this pluggable for airgapped clusters? or is making container `imagePullPolicy:IfNotPresent` sufficient?
		for imageID := range requiredSupportImages {
			supportFilesInitContainer := v1.Container{
				Name:         fmt.Sprintf("install-%s-support", imageID),
				Image:        fmt.Sprintf("gcr.io/gcp-dev-tools/duct-tape/%s", imageID),
				VolumeMounts: []v1.VolumeMount{supportVolumeMount},
			}
			podSpec.InitContainers = append(podSpec.InitContainers, supportFilesInitContainer)
		}
		// the populated volume is then mounted in the containers at `/dbg` too
		for _, container := range containersRequiringSupport {
			container.VolumeMounts = append(container.VolumeMounts, supportVolumeMount)
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

// allocatePort walks the podSpec's containers looking for an available port that is close to desiredPort.
// We deal with wrapping and avoid allocating ports < 1024
func allocatePort(podSpec *v1.PodSpec, desiredPort int32) int32 {
	var maxPort int32 = 65535 // ports are normally [1-65535]
	if desiredPort < 1024 || desiredPort > maxPort {
		desiredPort = 1024 // skip reserved ports
	}
	// We assume ports are rather sparsely allocated, so even if desiredPort
	// is allocated, desiredPort+1 or desiredPort+2 are likely to be free
	for port := desiredPort; port < maxPort; port++ {
		if isPortAvailable(podSpec, port) {
			return port
		}
	}
	for port := desiredPort; port > 1024; port-- {
		if isPortAvailable(podSpec, port) {
			return port
		}
	}
	panic("cannot find available port") // exceedingly unlikely
}

// isPortAvailable returns true if none of the pod's containers specify the given port.
func isPortAvailable(podSpec *v1.PodSpec, port int32) bool {
	for _, container := range podSpec.Containers {
		for _, portSpec := range container.Ports {
			if portSpec.ContainerPort == port {
				return false
			}
		}
	}
	return true
}

// transformContainer rewrites the container definition to enable debugging.
// Returns a debugging configuration description with associated language runtime support
// container image, or an error if the rewrite was unsuccessful.
func transformContainer(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (map[string]interface{}, string, error) {
	// update image configuration values with those set in the k8s manifest
	for _, envVar := range container.Env {
		// FIXME handle ValueFrom?
		if config.env == nil {
			config.env = make(map[string]string)
		}
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
			return transform.Apply(container, config, portAlloc), transform.RuntimeSupportImage(), nil
		}
	}
	return nil, "", errors.Errorf("unable to determine runtime for %q", container.Name)
}

func encodeConfigurations(configurations map[string]map[string]interface{}) string {
	bytes, err := json.Marshal(configurations)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func describe(obj runtime.Object) (group, version, kind, description string) {
	// get metadata/name; shamelessly stolen from from k8s.io/cli-runtime/pkg/printers/name.go
	name := "<unknown>"
	if acc, err := meta.Accessor(obj); err == nil {
		if n := acc.GetName(); len(n) > 0 {
			name = n
		}
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	group = gvk.Group
	version = gvk.Version
	kind = gvk.Kind
	if group == "" {
		description = fmt.Sprintf("%s/%s", strings.ToLower(kind), name)
	} else {
		description = fmt.Sprintf("%s.%s/%s", strings.ToLower(kind), group, name)
	}
	return
}
