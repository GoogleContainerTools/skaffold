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
These support files are provided through a set of support images defined at `gcr.io/k8s-skaffold/skaffold-debug-support/`
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

Each configuration is itself a JSON object of type `annotations.ContainerDebugConfiguration`, with an
`artifact` recording the corresponding artifact's `image` in the skaffold.yaml,
a `runtime` field identifying the language runtime, the working directory of the remote image (if known),
and a set of debugging ports.
*/
package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/annotations"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

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

const (
	// debuggingSupportVolume is the name of the volume used to hold language runtime debugging support files.
	debuggingSupportFilesVolume = "debugging-support-files"
)

// entrypointLaunchers is a list of known entrypoints that effectively just launches the container image's CMD
// as a command-line.  These entrypoints are ignored.
var entrypointLaunchers []string

var Protocols = []string{}

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
				log.Entry(context.TODO()).Errorf("deprecated versions not supported by debug: %s (%s)", description, version)
			} else {
				log.Entry(context.TODO()).Warnf("no debug transformation for: %s", description)
			}
		} else {
			log.Entry(context.TODO()).Debugf("no debug transformation for: %s", description)
		}
		return false
	}
}

// transformPodSpec attempts to configure a podspec for debugging.
// Returns true if changed, false otherwise.
func transformPodSpec(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, retrieveImageConfiguration configurationRetriever, debugHelpersRegistry string) bool {
	// order matters as rewriteProbes only affects containers marked for debugging
	containers := rewriteContainers(metadata, podSpec, retrieveImageConfiguration, debugHelpersRegistry)
	timeouts := rewriteProbes(metadata, podSpec)
	return containers || timeouts
}

// rewriteProbes rewrites k8s probes to expand timeouts to 10 minutes to allow debugging local probes.
func rewriteProbes(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec) bool {
	var minTimeout time.Duration = 10 * time.Minute // make it configurable?
	if annotation, found := metadata.Annotations[annotations.DebugProbeTimeouts]; found {
		if annotation == "skip" {
			log.Entry(context.TODO()).Debugf("skipping probe rewrite on %q by request", metadata.Name)
			return false
		}
		if d, err := time.ParseDuration(annotation); err != nil {
			log.Entry(context.TODO()).Warnf("invalid probe timeout value for %q: %q: %v", metadata.Name, annotation, err)
		} else {
			minTimeout = d
		}
	}
	annotation, found := metadata.Annotations[annotations.DebugConfig]
	if !found {
		log.Entry(context.TODO()).Debugf("skipping probe rewrite on %q: not configured for debugging", metadata.Name)
		return false
	}
	var config map[string]annotations.ContainerDebugConfiguration
	if err := json.Unmarshal([]byte(annotation), &config); err != nil {
		log.Entry(context.TODO()).Warnf("error unmarshalling debugging configuration for %q: %v", metadata.Name, err)
		return false
	}

	changed := false
	for i := range podSpec.Containers {
		c := &podSpec.Containers[i]
		// only affect containers listed in debug-config
		if _, found := config[c.Name]; found {
			lp := rewriteHTTPGetProbe(c.LivenessProbe, minTimeout)
			rp := rewriteHTTPGetProbe(c.ReadinessProbe, minTimeout)
			sp := rewriteHTTPGetProbe(c.StartupProbe, minTimeout)
			if lp || rp || sp {
				log.Entry(context.TODO()).Infof("Updated probe timeouts for %s/%s", metadata.Name, c.Name)
			}
			changed = changed || lp || rp || sp
		}
	}
	return changed
}

func rewriteHTTPGetProbe(probe *v1.Probe, minTimeout time.Duration) bool {
	if probe == nil || probe.HTTPGet == nil || int32(minTimeout.Seconds()) < probe.TimeoutSeconds {
		return false
	}
	probe.TimeoutSeconds = int32(minTimeout.Seconds())
	return true
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

func encodeConfigurations(configurations map[string]annotations.ContainerDebugConfiguration) string {
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
func exposePort(entries []containerPort, portName string, port int32) []containerPort {
	found := false
	for i := 0; i < len(entries); {
		switch {
		case entries[i].Name == portName:
			// Ports and names must be unique so rewrite an existing entry if found
			log.Entry(context.TODO()).Warnf("skaffold debug needs to expose port %d with name %s. Replacing clashing port definition %d (%s)", port, portName, entries[i].ContainerPort, entries[i].Name)
			entries[i].Name = portName
			entries[i].ContainerPort = port
			found = true
			i++
		case entries[i].ContainerPort == port:
			// Cut any entries with a clashing port
			log.Entry(context.TODO()).Warnf("skaffold debug needs to expose port %d for %s. Removing clashing port definition %d (%s)", port, portName, entries[i].ContainerPort, entries[i].Name)
			entries = append(entries[:i], entries[i+1:]...)
		default:
			i++
		}
	}
	if found {
		return entries
	}
	entry := containerPort{
		Name:          portName,
		ContainerPort: port,
	}
	return append(entries, entry)
}

func setEnvVar(entries containerEnv, key, value string) containerEnv {
	if _, found := entries.Env[key]; !found {
		entries.Order = append(entries.Order, key)
	}
	entries.Env[key] = value
	return entries
}

// shJoin joins the arguments into a quoted form suitable to pass to `sh -c`.
// Necessary as github.com/kballard/go-shellquote's `Join` quotes `$`.
func shJoin(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		if strings.ContainsAny(arg, " \t\r\n\"'()[]{}") {
			arg := strings.ReplaceAll(arg, `"`, `\"`)
			result += `"` + arg + `"`
		} else {
			result += arg
		}
	}
	return result
}
