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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type nodeTransformer struct{}

func init() {
	containerTransforms = append(containerTransforms, nodeTransformer{})
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
	return args[0] == "node" || strings.HasSuffix(args[0], "/node") ||
		args[0] == "nodemon" || strings.HasSuffix(args[0], "/nodemon")
}

// isLaunchingNpm determines if the arguments seems to be invoking npm
func isLaunchingNpm(args []string) bool {
	return args[0] == "npm" || strings.HasSuffix(args[0], "/npm")
}

func (t nodeTransformer) IsApplicable(config imageConfiguration) bool {
	if _, found := config.env["NODE_VERSION"]; found {
		return true
	}
	if len(config.entrypoint) > 0 {
		return isLaunchingNode(config.entrypoint) || isLaunchingNpm(config.entrypoint)
	} else if len(config.arguments) > 0 {
		return isLaunchingNode(config.arguments) || isLaunchingNpm(config.arguments)
	}
	return false
}

func (t nodeTransformer) RuntimeSupportImage() string {
	// no additional support required
	return ""
}

// Apply configures a container definition for NodeJS Chrome V8 Inspector.
// Returns a simple map describing the debug configuration details.
func (t nodeTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) map[string]interface{} {
	logrus.Infof("Configuring %q for node.js debugging", container.Name)

	// try to find existing `--inspect` command
	spec := retrieveNodeInspectSpec(config)
	// todo: find existing containerPort "devtools" and use port. But what if it conflicts with command-line spec?

	if spec == nil {
		spec = &inspectSpec{port: portAlloc(defaultDevtoolsPort)}
		switch {
		case len(config.entrypoint) > 0 && isLaunchingNode(config.entrypoint):
			container.Command = rewriteNodeCommandLine(config.entrypoint, *spec)

		case len(config.entrypoint) > 0 && isLaunchingNpm(config.entrypoint):
			container.Command = rewriteNpmCommandLine(config.entrypoint, *spec)

		case len(config.entrypoint) == 0 && len(config.arguments) > 0 && isLaunchingNode(config.arguments):
			container.Args = rewriteNodeCommandLine(config.arguments, *spec)

		case len(config.entrypoint) == 0 && len(config.arguments) > 0 && isLaunchingNpm(config.arguments):
			container.Args = rewriteNpmCommandLine(config.arguments, *spec)

		default:
			logrus.Warnf("Skipping %q as does not appear to invoke node", container.Name)
			return nil
		}
	}

	inspectPort := v1.ContainerPort{
		Name:          "devtools",
		ContainerPort: spec.port,
	}
	container.Ports = append(container.Ports, inspectPort)

	return map[string]interface{}{
		"runtime":  "nodejs",
		"devtools": spec.port,
	}
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
