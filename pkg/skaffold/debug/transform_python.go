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

type pythonTransformer struct{}

func init() {
	containerTransforms = append(containerTransforms, pythonTransformer{})
}

const (
	// most examples use 5678
	defaultPtvsdPort = 5678
)

// ptvsdSpec captures the useful python-ptvsd devtools options
type ptvsdSpec struct {
	host string
	port int32
	wait bool
}

// isLaunchingPython determines if the arguments seems to be invoking python
func isLaunchingPython(args []string) bool {
	return args[0] == "python" || strings.HasSuffix(args[0], "/python") ||
		args[0] == "python2" || strings.HasSuffix(args[0], "/python2") ||
		args[0] == "python3" || strings.HasSuffix(args[0], "/python3")
}

func (t pythonTransformer) IsApplicable(config imageConfiguration) bool {
	if _, found := config.env["PYTHON_VERSION"]; found {
		return true
	}
	if len(config.entrypoint) > 0 {
		return isLaunchingPython(config.entrypoint)
	} else if len(config.arguments) > 0 {
		return isLaunchingPython(config.arguments)
	}
	return false
}

func (t pythonTransformer) RuntimeSupportImage() string {
	return "python"
}

// Apply configures a container definition for Python with pydev/ptvsd
// Returns a simple map describing the debug configuration details.
func (t pythonTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) map[string]interface{} {
	logrus.Infof("Configuring %q for python debugging", container.Name)

	// try to find existing `-mptvsd` command
	spec := retrievePtvsdSpec(config)
	// todo: find existing containerPort "dap" (debug-adapter protocol) and use port. But what if it conflicts with command-line spec?

	if spec == nil {
		spec = &ptvsdSpec{host: "localhost", port: portAlloc(defaultPtvsdPort)}
		switch {
		case len(config.entrypoint) > 0 && isLaunchingPython(config.entrypoint):
			container.Command = rewritePythonCommandLine(config.entrypoint, *spec)

		case len(config.entrypoint) == 0 && len(config.arguments) > 0 && isLaunchingPython(config.arguments):
			container.Args = rewritePythonCommandLine(config.arguments, *spec)

		default:
			logrus.Warnf("Skipping %q as does not appear to invoke python", container.Name)
			return nil
		}
	}

	ptvsdPort := v1.ContainerPort{
		Name:          "dap", // debug adapter protocol
		ContainerPort: spec.port,
	}
	container.Ports = append(container.Ports, ptvsdPort)

	pythonUserBase := v1.EnvVar{
		Name:  "PYTHONUSERBASE",
		Value: "/dbg/python",
	}
	container.Env = append(container.Env, pythonUserBase)

	return map[string]interface{}{
		"runtime": "python",
		"dap":     spec.port,
	}
}

func retrievePtvsdSpec(config imageConfiguration) *ptvsdSpec {
	if spec := extractPtvsdArg(config.entrypoint); spec != nil {
		return spec
	}
	if spec := extractPtvsdArg(config.arguments); spec != nil {
		return spec
	}
	return nil
}

func extractPtvsdArg(args []string) *ptvsdSpec {
	if !hasPtvsdModule(args) {
		return nil
	}
	spec := ptvsdSpec{port: defaultPtvsdPort}
	for i, arg := range args {
		switch arg {
		case "--host":
			if i == len(args)-1 {
				return nil
			}
			spec.host = args[i+1]
		case "--port":
			if i == len(args)-1 {
				return nil
			}
			port, err := strconv.ParseInt(args[i+1], 10, 32)
			//spec.port, err := strconv.Atoi(args[i+1])
			if err != nil {
				logrus.Errorf("Invalid python ptvsd port %q: %s\n", args[i+1], err)
				return nil
			}
			spec.port = int32(port)

		case "--wait":
			spec.wait = true
		}
	}
	return &spec
}

func hasPtvsdModule(args []string) bool {
	if index := util.StrSliceIndex(args, "-mptvsd"); index >= 0 {
		return true
	}
	seenDashM := false
	for _, value := range args {
		if seenDashM {
			if value == "ptvsd" {
				return true
			}
			seenDashM = false
		} else if value == "-m" {
			seenDashM = true
		}
	}
	return false
}

// rewritePythonCommandLine rewrites a python command-line to insert a `-mptvsd` etc
func rewritePythonCommandLine(commandLine []string, spec ptvsdSpec) []string {
	// Assumes that commandLine[0] is "python" or "python3" etc
	return util.StrSliceInsert(commandLine, 1, spec.asArguments())
}

func (spec ptvsdSpec) asArguments() []string {
	args := []string{"-mptvsd"}
	if spec.host != "" {
		args = append(args, "--host", spec.host)
	}
	if spec.port >= 0 {
		args = append(args, "--port", strconv.FormatInt(int64(spec.port), 10))
	}
	if spec.wait {
		args = append(args, "--wait")
	}
	return args
}
