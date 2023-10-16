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

Each configuration is itself a JSON object of type `types.ContainerDebugConfiguration`, with an
`artifact` recording the corresponding artifact's `image` in the skaffold.yaml,
a `runtime` field identifying the language runtime, the working directory of the remote image (if known),
and a set of debugging ports.
*/
package debug

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// PortAllocator is a function that takes a desired port and returns an available port
// Ports are normally uint16 but Kubernetes types.ContainerPort is an integer
type PortAllocator func(int32) int32

// configurationRetriever retrieves an container image configuration
type ConfigurationRetriever func(string) (ImageConfiguration, error)

// ImageConfiguration captures information from a docker/oci image configuration.
// It also includes a "artifact", usually containing the corresponding artifact's' image name from `skaffold.yaml`.
type ImageConfiguration struct {
	// Artifact is the corresponding Artifact's image name (`pkg/skaffold/build.Artifact.ImageName`)
	Artifact    string
	RuntimeType types.Runtime
	Author      string
	Labels      map[string]string
	Env         map[string]string
	Entrypoint  []string
	Arguments   []string
	WorkingDir  string
}

const (
	// DebuggingSupportVolume is the name of the volume used to hold language runtime debugging support files.
	DebuggingSupportFilesVolume = "debugging-support-files"
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

func EncodeConfigurations(configurations map[string]types.ContainerDebugConfiguration) string {
	bytes, err := json.Marshal(configurations)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// exposePort adds a `types.ContainerPort` instance or amends an existing entry with the same port.
func exposePort(entries []types.ContainerPort, portName string, port int32) []types.ContainerPort {
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
	entry := types.ContainerPort{
		Name:          portName,
		ContainerPort: port,
	}
	return append(entries, entry)
}

func setEnvVar(entries types.ContainerEnv, key, value string) types.ContainerEnv {
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
