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
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
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
	return len(args) > 0 &&
		(args[0] == "python" || strings.HasSuffix(args[0], "/python") ||
			args[0] == "python2" || strings.HasSuffix(args[0], "/python2") ||
			args[0] == "python3" || strings.HasSuffix(args[0], "/python3"))
}

func (t pythonTransformer) IsApplicable(config imageConfiguration) bool {
	// We can only put Python in debug mode by modifying the python command line,
	// so looking for Python-related environment variables is insufficient.
	if len(config.entrypoint) > 0 && !isEntrypointLauncher(config.entrypoint) {
		return isLaunchingPython(config.entrypoint)
	}
	return isLaunchingPython(config.arguments)
}

// Apply configures a container definition for Python with pydev/ptvsd
// Returns a simple map describing the debug configuration details.
func (t pythonTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (ContainerDebugConfiguration, string, error) {
	logrus.Infof("Configuring %q for python debugging", container.Name)

	// try to find existing `-mptvsd` command
	spec := retrievePtvsdSpec(config)

	if spec == nil {
		spec = &ptvsdSpec{port: portAlloc(defaultPtvsdPort)}
		switch {
		case isLaunchingPython(config.entrypoint):
			container.Command = rewritePythonCommandLine(config.entrypoint, *spec)

		case (len(config.entrypoint) == 0 || isEntrypointLauncher(config.entrypoint)) && isLaunchingPython(config.arguments):
			container.Args = rewritePythonCommandLine(config.arguments, *spec)

		default:
			return ContainerDebugConfiguration{}, "", fmt.Errorf("%q does not appear to invoke python", container.Name)
		}
	}

	pyUserBase := "/dbg/python"
	if existing, found := config.env["PYTHONUSERBASE"]; found {
		// todo: handle windows containers?
		pyUserBase = pyUserBase + ":" + existing
	}
	container.Env = setEnvVar(container.Env, "PYTHONUSERBASE", pyUserBase)
	container.Ports = exposePort(container.Ports, "dap", spec.port)

	return ContainerDebugConfiguration{
		Runtime: "python",
		Ports:   map[string]uint32{"dap": uint32(spec.port)},
	}, "python", nil
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
			// spec.port, err := strconv.Atoi(args[i+1])
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
	// --host is a mandatory argument
	if spec.host == "" {
		args = append(args, "--host", "0.0.0.0")
	} else {
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
