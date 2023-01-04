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

package debug

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

type jdwpTransformer struct{}

//nolint:golint
func NewJDWPTransformer() containerTransformer {
	return jdwpTransformer{}
}

func init() {
	RegisterContainerTransformer(NewJDWPTransformer())
}

const (
	// no standard port for JDWP; most examples use 5005 or 8000
	defaultJdwpPort = 5005
)

func (t jdwpTransformer) IsApplicable(config ImageConfiguration) bool {
	if config.RuntimeType == types.Runtimes.JVM {
		log.Entry(context.TODO()).Infof("Artifact %q has JVM runtime: specified by user in skaffold config", config.Artifact)
		return true
	}
	if _, found := config.Env["JAVA_TOOL_OPTIONS"]; found {
		return true
	}
	if _, found := config.Env["JAVA_VERSION"]; found {
		return true
	}
	if len(config.Entrypoint) > 0 && !isEntrypointLauncher(config.Entrypoint) {
		return config.Entrypoint[0] == "java" || strings.HasSuffix(config.Entrypoint[0], "/java")
	}
	return len(config.Arguments) > 0 &&
		(config.Arguments[0] == "java" || strings.HasSuffix(config.Arguments[0], "/java"))
}

// captures the useful jdwp options (see `java -agentlib:jdwp=help`)
type jdwpSpec struct {
	transport string
	// `address` portion is split into host/port
	host    string
	port    uint16
	quiet   bool
	suspend bool
	server  bool
}

// Apply configures a container definition for JVM debugging.
// Returns a simple map describing the debug configuration details.
func (t jdwpTransformer) Apply(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator, overrideProtocols []string) (types.ContainerDebugConfiguration, string, error) {
	container := adapter.GetContainer()
	log.Entry(context.TODO()).Infof("Configuring %q for JVM debugging", container.Name)
	// try to find existing JAVA_TOOL_OPTIONS or jdwp command argument
	spec := retrieveJdwpSpec(config)

	var port int32
	if spec != nil {
		port = int32(spec.port)
	} else {
		port = portAlloc(defaultJdwpPort)
		jto := fmt.Sprintf("-agentlib:jdwp=transport=dt_socket,server=y,address=%d,suspend=n,quiet=y", port)
		if existing, found := config.Env["JAVA_TOOL_OPTIONS"]; found {
			jto = existing + " " + jto
		}
		container.Env = setEnvVar(container.Env, "JAVA_TOOL_OPTIONS", jto)
	}

	container.Ports = exposePort(container.Ports, "jdwp", port)

	return types.ContainerDebugConfiguration{
		Runtime: "jvm",
		Ports:   map[string]uint32{"jdwp": uint32(port)},
	}, "", nil
}

func retrieveJdwpSpec(config ImageConfiguration) *jdwpSpec {
	for _, arg := range config.Entrypoint {
		if spec := extractJdwpArg(arg); spec != nil {
			return spec
		}
	}
	for _, arg := range config.Arguments {
		if spec := extractJdwpArg(arg); spec != nil {
			return spec
		}
	}
	// Nobody should be setting JDWP options via _JAVA_OPTIONS and IBM_JAVA_OPTIONS
	if value, found := config.Env["JAVA_TOOL_OPTIONS"]; found {
		for _, arg := range strings.Split(value, " ") {
			if spec := extractJdwpArg(arg); spec != nil {
				return spec
			}
		}
	}
	return nil
}

func extractJdwpArg(spec string) *jdwpSpec {
	if strings.Index(spec, "-agentlib:jdwp=") == 0 {
		return parseJdwpSpec(spec[15:])
	}
	if strings.Index(spec, "-Xrunjdwp:") == 0 {
		return parseJdwpSpec(spec[10:])
	}
	return nil
}

func (spec jdwpSpec) String() string {
	result := []string{"transport=" + spec.transport}
	if spec.quiet {
		result = append(result, "quiet=y")
	}
	if spec.server {
		result = append(result, "server=y")
	}
	if !spec.suspend {
		result = append(result, "suspend=n")
	}
	if spec.port > 0 {
		if len(spec.host) > 0 {
			result = append(result, "address="+spec.host+":"+strconv.FormatUint(uint64(spec.port), 10))
		} else {
			result = append(result, "address="+strconv.FormatUint(uint64(spec.port), 10))
		}
	}
	return strings.Join(result, ",")
}

// parseJdwpSpec parses a JDWP spec string as passed to `-agentlib:jdwp=` or `-Xrunjdwp:`
// like `transport=dt_socket,server=y,address=8000,quiet=y,suspend=n`
func parseJdwpSpec(specification string) *jdwpSpec {
	parsed := make(map[string]string)
	for _, component := range strings.Split(specification, ",") {
		if len(component) > 0 {
			keyValue := strings.SplitN(component, "=", 2)
			if len(keyValue) == 2 {
				parsed[keyValue[0]] = keyValue[1]
			}
			// else return error?
		}
	}
	// use defaults as per https://docs.oracle.com/javase/7/docs/technotes/guides/jpda/conninv.html#jdwpoptions
	spec := jdwpSpec{
		transport: "dt_socket",
		quiet:     false,
		suspend:   true,
		server:    false,
		host:      "",
		port:      0,
	}
	if transport, found := parsed["transport"]; found {
		spec.transport = transport
	}
	if quietYN, found := parsed["quiet"]; found {
		spec.quiet = quietYN == "y"
	}
	if suspendYN, found := parsed["suspend"]; found {
		spec.suspend = suspendYN == "y"
	}
	if serverYN, found := parsed["server"]; found {
		spec.server = serverYN == "y"
	}
	if address, found := parsed["address"]; found {
		split := strings.SplitN(address, ":", 2)
		switch len(split) {
		// port only
		case 1:
			p, _ := strconv.ParseUint(split[0], 10, 16)
			spec.port = uint16(p)

		// host and port
		case 2:
			spec.host = split[0]
			p, _ := strconv.ParseUint(split[1], 10, 16)
			spec.port = uint16(p)
		}
	}
	return &spec
}
