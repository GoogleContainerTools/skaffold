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
Package debug transforms Kubernetes pod-bearing resources so as to configure containers
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
    "microservice":{"artifact":"node-example","runtime":"nodejs","ports":{"devtools":9229}},
    "adapter":{"artifact":"java-example","runtime":"jvm","ports":{"jdwp":5005}}
  }'

Each configuration is itself a JSON object of type `ContainerDebugConfiguration`, with an
`artifact` recording the corresponding artifact's `image` in the skaffold.yaml,
a `runtime` field identifying the language runtime, the working directory of the remote image (if known),
and a set of debugging ports.
*/
package debug

import (
	"encoding/json"
	"fmt"
	"strings"

	shell "github.com/kballard/go-shellquote"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ContainerDebugConfiguration captures debugging information for a specific container.
// This structure is serialized out and included in the pod metadata.
type ContainerDebugConfiguration struct {
	// Artifact is the corresponding artifact's image name used in the skaffold.yaml
	Artifact string `json:"artifact,omitempty"`
	// Runtime represents the underlying language runtime (`go`, `jvm`, `nodejs`, `python`)
	Runtime string `json:"runtime,omitempty"`
	// WorkingDir is the working directory in the image configuration; may be empty
	WorkingDir string `json:"workingDir,omitempty"`
	// Ports is the list of debugging ports, keyed by protocol type
	Ports map[string]uint32 `json:"ports,omitempty"`
}

// portAllocator is a function that takes a desired port and returns an available port
// Ports are normally uint16 but Kubernetes ContainerPort.containerPort is an integer
type portAllocator func(int32) int32

// configurationRetriever retrieves an container image configuration
type configurationRetriever func(string) (imageConfiguration, error)

// imageConfiguration captures information from a docker/oci image configuration.
// It also includes a "artifact", usually containing the corresponding artifact's' image name from `skaffold.yaml`.
type imageConfiguration struct {
	// artifact is the corresponding artifact's image name (`pkg/skaffold/build.Artifact.ImageName`)
	artifact string

	labels     map[string]string
	env        map[string]string
	entrypoint []string
	arguments  []string
	workingDir string
}

// containerTransformer transforms a container definition
type containerTransformer interface {
	// IsApplicable determines if this container is suitable to be transformed.
	IsApplicable(config imageConfiguration) bool

	// Apply configures a container definition for debugging, returning the debug configuration details
	// and required initContainer (an empty string if not required), or return a non-nil error if
	// the container could not be transformed.  The initContainer image is intended to install any
	// required debug support tools.
	Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (ContainerDebugConfiguration, string, error)
}

const (
	// debuggingSupportVolume is the name of the volume used to hold language runtime debugging support files.
	debuggingSupportFilesVolume = "debugging-support-files"

	// DebugConfigAnnotation is the name of the podspec annotation that records debugging configuration information.
	DebugConfigAnnotation = "debug.cloud.google.com/config"
)

// containerTransforms are the set of configured transformers
var containerTransforms []containerTransformer

// entrypointLaunchers is a list of known entrypoints that effectively just launches the container image's CMD
// as a command-line.  These entrypoints are ignored.
var entrypointLaunchers []string

// isEntrypointLauncher checks if the given entrypoint is a known entrypoint launcher,
// meaning an entrypoint that treats the image's CMD as a command-line.
func isEntrypointLauncher(entrypoint []string) bool {
	if len(entrypoint) != 1 {
		return false
	}
	for _, knownEntrypoints := range entrypointLaunchers {
		if knownEntrypoints == entrypoint[0] {
			return true
		}
	}
	return false
}

// transformManifest attempts to configure a manifest for debugging.
// Returns true if changed, false otherwise.
func transformManifest(obj runtime.Object, retrieveImageConfiguration configurationRetriever, debugHelpersRegistry string) bool {
	one := int32(1)
	switch o := obj.(type) {
	case *v1.Pod:
		return transformPodSpec(&o.ObjectMeta, &o.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *v1.PodList:
		changed := false
		for i := range o.Items {
			if transformPodSpec(&o.Items[i].ObjectMeta, &o.Items[i].Spec, retrieveImageConfiguration, debugHelpersRegistry) {
				changed = true
			}
		}
		return changed
	case *v1.ReplicationController:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.Deployment:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.DaemonSet:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.ReplicaSet:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.StatefulSet:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *batchv1.Job:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)

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
func transformPodSpec(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, retrieveImageConfiguration configurationRetriever, debugHelpersRegistry string) bool {
	// skip annotated podspecs — allows users to customize their own image
	if _, found := metadata.Annotations[DebugConfigAnnotation]; found {
		return false
	}

	portAlloc := func(desiredPort int32) int32 {
		return allocatePort(podSpec, desiredPort)
	}
	// map of containers -> debugging configuration maps; k8s ensures that a pod's containers are uniquely named
	configurations := make(map[string]ContainerDebugConfiguration)
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
		// `err != nil` means that the container did not or could not be transformed
		if configuration, requiredImage, err := transformContainer(container, imageConfig, portAlloc); err == nil {
			configuration.Artifact = imageConfig.artifact
			if configuration.WorkingDir == "" {
				configuration.WorkingDir = imageConfig.workingDir
			}
			configurations[container.Name] = configuration
			if len(requiredImage) > 0 {
				logrus.Infof("%q requires debugging support image %q", container.Name, requiredImage)
				containersRequiringSupport = append(containersRequiringSupport, container)
				requiredSupportImages[requiredImage] = true
			}
			// todo: add this artifact to the watch list?
		} else {
			logrus.Warnf("Image %q not configured for debugging: %v", container.Name, err)
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
		for imageID := range requiredSupportImages {
			supportFilesInitContainer := v1.Container{
				Name:         fmt.Sprintf("install-%s-support", imageID),
				Image:        fmt.Sprintf("%s/%s", debugHelpersRegistry, imageID),
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
		metadata.Annotations[DebugConfigAnnotation] = encodeConfigurations(configurations)
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
func transformContainer(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (ContainerDebugConfiguration, string, error) {
	// Update the image configuration's environment with those set in the k8s manifest.
	// (Environment variables in the k8s container's `env` add to the image configuration's `env` settings rather than replace.)
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

	// Apply command-line unwrapping for buildpack images and images using `sh -c`-style command-lines
	next := func(container *v1.Container, config imageConfiguration) (ContainerDebugConfiguration, string, error) {
		return performContainerTransform(container, config, portAlloc)
	}
	if _, found := config.labels["io.buildpacks.stack.id"]; found && len(config.entrypoint) > 0 && config.entrypoint[0] == "/cnb/lifecycle/launcher" {
		return updateForCNBImage(container, config, next)
	}
	return updateForShDashC(container, config, next)
}

func updateForShDashC(container *v1.Container, ic imageConfiguration, transformer func(*v1.Container, imageConfiguration) (ContainerDebugConfiguration, string, error)) (ContainerDebugConfiguration, string, error) {
	var rewriter func([]string)
	copy := ic
	switch {
	// Case 1: entrypoint = ["/bin/sh", "-c"], arguments = ["<cmd-line>", args ...]
	case len(ic.entrypoint) == 2 && len(ic.arguments) > 0 && isShDashC(ic.entrypoint[0], ic.entrypoint[1]):
		if split, err := shell.Split(ic.arguments[0]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
				container.Command = nil // inherit from container
				container.Args = append([]string{shJoin(rewrite)}, ic.arguments[1:]...)
			}
		}

	// Case 2: entrypoint = ["/bin/sh", "-c", "<cmd-line>", args...], arguments = [args ...]
	case len(ic.entrypoint) > 2 && isShDashC(ic.entrypoint[0], ic.entrypoint[1]):
		if split, err := shell.Split(ic.entrypoint[2]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
				container.Command = append([]string{ic.entrypoint[0], ic.entrypoint[1], shJoin(rewrite)}, ic.entrypoint[3:]...)
			}
		}

	// Case 3: entrypoint = [] or an entrypoint launcher (and so ignored), arguments = ["/bin/sh", "-c", "<cmd-line>", args...]
	case (len(ic.entrypoint) == 0 || isEntrypointLauncher(ic.entrypoint)) && len(ic.arguments) > 2 && isShDashC(ic.arguments[0], ic.arguments[1]):
		if split, err := shell.Split(ic.arguments[2]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
				container.Command = nil
				container.Args = append([]string{ic.arguments[0], ic.arguments[1], shJoin(rewrite)}, ic.arguments[3:]...)
			}
		}
	}

	c, image, err := transformer(container, copy)
	if err == nil && rewriter != nil && container.Command != nil {
		rewriter(container.Command)
	}
	return c, image, err
}

func isShDashC(cmd, arg string) bool {
	return (cmd == "/bin/sh" || cmd == "/bin/bash") && arg == "-c"
}

func performContainerTransform(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (ContainerDebugConfiguration, string, error) {
	for _, transform := range containerTransforms {
		if transform.IsApplicable(config) {
			return transform.Apply(container, config, portAlloc)
		}
	}
	return ContainerDebugConfiguration{}, "", fmt.Errorf("unable to determine runtime for %q", container.Name)
}

func encodeConfigurations(configurations map[string]ContainerDebugConfiguration) string {
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

// exposePort adds a `ContainerPort` instance or amends an existing entry with the same port.
func exposePort(entries []v1.ContainerPort, portName string, port int32) []v1.ContainerPort {
	found := false
	for i := 0; i < len(entries); {
		switch {
		case entries[i].Name == portName:
			// Ports and names must be unique so rewrite an existing entry if found
			logrus.Warnf("skaffold debug needs to expose port %d with name %s. Replacing clashing port definition %d (%s)", port, portName, entries[i].ContainerPort, entries[i].Name)
			entries[i].Name = portName
			entries[i].ContainerPort = port
			found = true
			i++
		case entries[i].ContainerPort == port:
			// Cut any entries with a clashing port
			logrus.Warnf("skaffold debug needs to expose port %d for %s. Removing clashing port definition %d (%s)", port, portName, entries[i].ContainerPort, entries[i].Name)
			entries = append(entries[:i], entries[i+1:]...)
		default:
			i++
		}
	}
	if found {
		return entries
	}
	entry := v1.ContainerPort{
		Name:          portName,
		ContainerPort: port,
	}
	return append(entries, entry)
}

// setEnvVar adds a `EnvVar` instance or replaced an existing entry
func setEnvVar(entries []v1.EnvVar, varName, value string) []v1.EnvVar {
	for i := range entries {
		// env variable names must be unique so rewrite an existing entry if found
		if entries[i].Name == varName {
			entries[i].Value = value
			return entries
		}
	}

	entry := v1.EnvVar{
		Name:  varName,
		Value: value,
	}
	return append(entries, entry)
}

// shJoin joins the arguments into a quoted form suitable to pass to `sh -c`.
// Necessary as github.com/kballard/go-shellquote's `Join` quotes `$`.
func shJoin(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		if strings.ContainsAny(arg, " \t\r\n\"") {
			arg := strings.ReplaceAll(arg, `"`, `\"`)
			result += `"` + arg + `"`
		} else {
			result += arg
		}
	}
	return result
}
