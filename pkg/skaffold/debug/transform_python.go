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
	defaultPtvsdPort   = 5678
	defaultDebugpyPort = 5678
)

type pythonDebugType int

const (
	ptvsd pythonDebugType = iota
	debugpy
)

// pythonSpec captures the useful python-ptvsd devtools options
type pythonSpec struct {
	debugger pythonDebugType
	host     string
	port     int32
	wait     bool
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

// Apply configures a container definition for Python with ptvsd/debugpy.
// Returns a simple map describing the debug configuration details.
func (t pythonTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (ContainerDebugConfiguration, string, error) {
	logrus.Infof("Configuring %q for python debugging", container.Name)

	// try to find existing `-mptvsd` or `-mdebugpy` command
	if spec := retrievePythonDebugSpec(config); spec != nil {
		container.Ports = exposePort(container.Ports, "dap", spec.port)
		return ContainerDebugConfiguration{
			Runtime: "python",
			Ports:   map[string]uint32{"dap": uint32(spec.port)},
		}, "", nil
	}

	spec := &pythonSpec{debugger: debugpy, port: portAlloc(defaultDebugpyPort)}
	switch {
	case isLaunchingPython(config.entrypoint):
		container.Command = rewritePythonCommandLine(config.entrypoint, *spec)

	case (len(config.entrypoint) == 0 || isEntrypointLauncher(config.entrypoint)) && isLaunchingPython(config.arguments):
		container.Args = rewritePythonCommandLine(config.arguments, *spec)

	default:
		return ContainerDebugConfiguration{}, "", fmt.Errorf("%q does not appear to invoke python", container.Name)
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

func retrievePythonDebugSpec(config imageConfiguration) *pythonSpec {
	if spec := extractPythonDebugSpec(config.entrypoint); spec != nil {
		return spec
	}
	if spec := extractPythonDebugSpec(config.arguments); spec != nil {
		return spec
	}
	return nil
}

func extractPythonDebugSpec(args []string) *pythonSpec {
	if spec := extractPtvsdSpec(args); spec != nil {
		return spec
	}
	if spec := extractDebugpySpec(args); spec != nil {
		return spec
	}
	return nil
}

func extractPtvsdSpec(args []string) *pythonSpec {
	if !hasPyModule("ptvsd", args) {
		return nil
	}
	spec := pythonSpec{debugger: ptvsd, port: defaultPtvsdPort}
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

func extractDebugpySpec(args []string) *pythonSpec {
	if !hasPyModule("debugpy", args) {
		return nil
	}
	spec := pythonSpec{debugger: debugpy, port: -1}
	for i, arg := range args {
		switch arg {
		case "--listen":
			if i == len(args)-1 {
				return nil
			}
			s := strings.SplitN(args[i+1], ":", 2)
			if len(s) > 1 {
				spec.host = s[0]
			}
			port, err := strconv.ParseInt(s[len(s)-1], 10, 32)
			if err != nil {
				logrus.Errorf("Invalid port %q: %s\n", args[i+1], err)
				return nil
			}
			spec.port = int32(port)

		case "--wait-for-client":
			spec.wait = true
		}
	}
	if spec.port < 0 {
		return nil
	}
	return &spec
}

func hasPyModule(module string, args []string) bool {
	if index := util.StrSliceIndex(args, "-m"+module); index >= 0 {
		return true
	}
	seenDashM := false
	for _, value := range args {
		if seenDashM {
			if value == module {
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
func rewritePythonCommandLine(commandLine []string, spec pythonSpec) []string {
	// Assumes that commandLine[0] is "python" or "python3" etc
	return util.StrSliceInsert(commandLine, 1, spec.asArguments())
}

func (spec pythonSpec) asArguments() []string {
	switch spec.debugger {
	case ptvsd:
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

	case debugpy:
		args := []string{"-mdebugpy"}

		if spec.host == "" {
			args = append(args, "--listen", strconv.FormatInt(int64(spec.port), 10))
		} else {
			args = append(args, "--listen", fmt.Sprintf("%s:%d", spec.host, spec.port))
		}
		if spec.wait {
			args = append(args, "--wait-for-client")
		}
		return args
	}
	return nil
}
