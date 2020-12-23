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
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type nodeTransformer struct{}

func init() {
	containerTransforms = append(containerTransforms, nodeTransformer{})

	// the `node` image's "docker-entrypoint.sh" launches the command
	entrypointLaunchers = append(entrypointLaunchers, "docker-entrypoint.sh")
}

const (
	// most examples use 9229
	defaultDevtoolsPort = 9229
)

// inspectSpec captures the useful nodejs devtools options
type inspectSpec struct {
	host string
	port int32
	brk  bool
}

// isLaunchingNode determines if the arguments seems to be invoking node
func isLaunchingNode(args []string) bool {
	return len(args) > 0 && (args[0] == "node" || strings.HasSuffix(args[0], "/node") ||
		args[0] == "nodemon" || strings.HasSuffix(args[0], "/nodemon"))
}

// isLaunchingNpm determines if the arguments seems to be invoking npm
func isLaunchingNpm(args []string) bool {
	return len(args) > 0 && (args[0] == "npm" || strings.HasSuffix(args[0], "/npm"))
}

func (t nodeTransformer) IsApplicable(config imageConfiguration) bool {
	// NODE_VERSION defined in Official Docker `node` image
	// NODEJS_VERSION defined in RedHat's node base image
	// NODE_ENV is a common var found to toggle debug and production
	for _, v := range []string{"NODE_VERSION", "NODEJS_VERSION", "NODE_ENV"} {
		if _, found := config.env[v]; found {
			return true
		}
	}
	if len(config.entrypoint) > 0 && !isEntrypointLauncher(config.entrypoint) {
		return isLaunchingNode(config.entrypoint) || isLaunchingNpm(config.entrypoint)
	}
	return isLaunchingNode(config.arguments) || isLaunchingNpm(config.arguments)
}

// Apply configures a container definition for NodeJS Chrome V8 Inspector.
// Returns a simple map describing the debug configuration details.
func (t nodeTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (ContainerDebugConfiguration, string, error) {
	logrus.Infof("Configuring %q for node.js debugging", container.Name)

	// try to find existing `--inspect` command
	spec := retrieveNodeInspectSpec(config)
	if spec == nil {
		spec = &inspectSpec{host: "0.0.0.0", port: portAlloc(defaultDevtoolsPort)}
		switch {
		case isLaunchingNode(config.entrypoint):
			container.Command = rewriteNodeCommandLine(config.entrypoint, *spec)

		case isLaunchingNpm(config.entrypoint):
			container.Command = rewriteNpmCommandLine(config.entrypoint, *spec)

		case (len(config.entrypoint) == 0 || isEntrypointLauncher(config.entrypoint)) && isLaunchingNode(config.arguments):
			container.Args = rewriteNodeCommandLine(config.arguments, *spec)

		case (len(config.entrypoint) == 0 || isEntrypointLauncher(config.entrypoint)) && isLaunchingNpm(config.arguments):
			container.Args = rewriteNpmCommandLine(config.arguments, *spec)

		default:
			if v, found := config.env["NODE_OPTIONS"]; found {
				container.Env = setEnvVar(container.Env, "NODE_OPTIONS", v+" "+spec.String())
			} else {
				container.Env = setEnvVar(container.Env, "NODE_OPTIONS", spec.String())
			}
		}
	}

	// Add our debug-helper path to resolve to our node wrapper
	if v, found := config.env["PATH"]; found {
		container.Env = setEnvVar(container.Env, "PATH", "/dbg/nodejs/bin:"+v)
	} else {
		container.Env = setEnvVar(container.Env, "PATH", "/dbg/nodejs/bin")
	}

	container.Ports = exposePort(container.Ports, "devtools", spec.port)

	return ContainerDebugConfiguration{
		Runtime: "nodejs",
		Ports:   map[string]uint32{"devtools": uint32(spec.port)},
	}, "nodejs", nil
}

func retrieveNodeInspectSpec(config imageConfiguration) *inspectSpec {
	for _, arg := range config.entrypoint {
		if spec := extractInspectArg(arg); spec != nil {
			return spec
		}
	}
	for _, arg := range config.arguments {
		if spec := extractInspectArg(arg); spec != nil {
			return spec
		}
	}
	if value, found := config.env["NODE_OPTIONS"]; found {
		if spec := extractInspectArg(value); spec != nil {
			return spec
		}
	}
	return nil
}

// extractInspectArg attempts to parse out an `--inspect=xxx` argument,
// returning true if found and false if not
func extractInspectArg(arg string) *inspectSpec {
	spec := inspectSpec{port: 9229}
	address := ""
	switch {
	case strings.Index(arg, "--inspect=") == 0:
		address = arg[10:]
		fallthrough
	case arg == "--inspect":
		spec.brk = false

	case strings.Index(arg, "--inspect-brk=") == 0:
		address = arg[14:]
		fallthrough
	case arg == "--inspect-brk":
		spec.brk = true

	default:
		return nil
	}
	if len(address) > 0 {
		if split := strings.SplitN(address, ":", 2); len(split) == 1 {
			port, err := strconv.ParseInt(split[0], 10, 32)
			if err != nil {
				logrus.Errorf("Invalid NodeJS inspect port %q: %s\n", address, err)
				return nil
			}
			spec.port = int32(port)
		} else {
			spec.host = split[0]
			port, err := strconv.ParseInt(split[1], 10, 32)
			if err != nil {
				logrus.Errorf("Invalid NodeJS inspect port %q: %s\n", address, err)
				return nil
			}
			spec.port = int32(port)
		}
	}
	return &spec
}

func (spec inspectSpec) String() string {
	s := "--inspect"
	if spec.brk {
		s = "--inspect-brk"
	}
	if len(spec.host) > 0 {
		s += "=" + spec.host + ":" + strconv.FormatInt(int64(spec.port), 10)
	} else if spec.port > 0 {
		s += "=" + strconv.FormatInt(int64(spec.port), 10)
	}
	return s
}

// rewriteNodeCommandLine rewrites a node/nodemon command-line to insert a `--inspect=xxx`
func rewriteNodeCommandLine(commandLine []string, spec inspectSpec) []string {
	// Assumes that commandLine[0] is "node" or "nodemon"
	commandLine = append(commandLine, "")
	copy(commandLine[2:], commandLine[1:]) // shift
	commandLine[1] = spec.String()
	return commandLine
}

// rewriteNpmCommandLine rewrites an npm command-line to add a `--node-options=--inspect=xxx`
func rewriteNpmCommandLine(commandLine []string, spec inspectSpec) []string {
	// Assumes that commandLine[0] is "npm"
	newOption := "--node-options=" + spec.String()
	// see if there is "--" for end of npm arguments
	if index := util.StrSliceIndex(commandLine, "--"); index > 0 {
		commandLine = append(commandLine, "")
		copy(commandLine[index+1:], commandLine[index:]) // shift
		commandLine[index] = newOption
	} else {
		commandLine = append(commandLine, newOption)
	}
	return commandLine
}
