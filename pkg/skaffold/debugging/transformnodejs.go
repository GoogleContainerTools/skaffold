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
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type nodeTransformer struct{}

func init() {
	containerTransforms = append(containerTransforms, nodeTransformer{})
}

// captures the useful nodejs devtools options
type inspectSpec struct {
	host string
	port int32
	brk  bool
}

func (t nodeTransformer) IsApplicable(config imageConfiguration) bool {
	if _, found := config.env["NODE_VERSION"]; found {
		return true
	}
	if len(config.entrypoint) > 0 {
		if config.entrypoint[0] == "node" || strings.HasSuffix(config.entrypoint[0], "/node") ||
			config.entrypoint[0] == "nodemon" || strings.HasSuffix(config.entrypoint[0], "/nodemon") {
			return true
		}
	} else if len(config.arguments) > 0 {
		if config.arguments[0] == "node" || strings.HasSuffix(config.arguments[0], "/node") ||
			config.arguments[0] == "nodemon" || strings.HasSuffix(config.arguments[0], "/nodemon") {
			return true
		}
	}
	return false
}

// configureNodeJsDebugging configures a container definition for NodeJS Chrome V8 Inspector.
// Returns a simple map describing the debug configuration details.
func (t nodeTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) map[string]interface{} {
	logrus.Infof("Configuring [%s] for node.js debugging", container.Name)

	// try to find existing `--inspect` command
	spec := retrieveNodeInspectSpec(config)
	// todo: find existing containerPort "devtools" and use port. But what if it conflicts with command-line spec?

	if spec == nil {
		// most examples use 9229
		spec = &inspectSpec{port: portAlloc(9229)}
		if len(config.entrypoint) > 0 && (config.entrypoint[0] == "node" || strings.HasSuffix(config.entrypoint[0], "/node") ||
			config.entrypoint[0] == "nodemon" || strings.HasSuffix(config.entrypoint[0], "/nodemon")) {
			container.Command = append(config.entrypoint, "")
			copy(container.Command[2:], container.Command[1:])
			container.Command[1] = spec.String()
		} else if len(config.entrypoint) == 0 && len(config.arguments) > 0 && (config.arguments[0] == "node" || strings.HasSuffix(config.arguments[0], "/node") ||
			config.arguments[0] == "nodemon" || strings.HasSuffix(config.arguments[0], "/nodemon")) {
			container.Args = append(config.arguments, "")
			copy(container.Args[2:], container.Args[1:])
			container.Args[1] = spec.String()
		} else {
			logrus.Warn("Skipping as does not appear to invoke node")
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
	return nil
}

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
				logrus.Errorf("Invalid NodeJS inspect port \"%s\": %s\n", address, err)
				return nil
			}
			spec.port = int32(port)
		} else {
			spec.host = split[0]
			port, err := strconv.ParseInt(split[1], 10, 32)
			if err != nil {
				logrus.Errorf("Invalid NodeJS inspect port \"%s\": %s\n", address, err)
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

