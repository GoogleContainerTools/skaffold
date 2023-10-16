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
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
)

type pythonTransformer struct{}

//nolint:golint
func NewPythonTransformer() containerTransformer {
	return pythonTransformer{}
}

func init() {
	RegisterContainerTransformer(NewPythonTransformer())
}

const (
	// most examples use 5678
	defaultPtvsdPort   = 5678
	defaultDebugpyPort = 5678
	defaultPydevdPort  = 5678
)

type pythonDebugType int

const (
	ptvsd pythonDebugType = iota
	debugpy
	pydevd
)

const (
	pydevdProtocol = "pydevd"
	dapProtocol    = "dap"
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
	return len(args) > 0 && (args[0] == "python" || strings.HasSuffix(args[0], "/python") ||
		args[0] == "python2" || strings.HasSuffix(args[0], "/python2") ||
		args[0] == "python3" || strings.HasSuffix(args[0], "/python3"))
}

func hasCommonPythonEnvVars(env map[string]string) bool {
	for _, key := range []string{
		"PYTHON_VERSION",
		"PYTHONVERBOSE",
		"PYTHONINSPECT",
		"PYTHONOPTIMIZE",
		"PYTHONUSERSITE",
		"PYTHONUNBUFFERED",
		"PYTHONPATH",
		"PYTHONUSERBASE",
		"PYTHONWARNINGS",
		"PYTHONHOME",
		"PYTHONCASEOK",
		"PYTHONIOENCODING",
		"PYTHONHASHSEED",
		"PYTHONDONTWRITEBYTECODE",
	} {
		if _, found := env[key]; found {
			return true
		}
	}

	return false
}

func (t pythonTransformer) IsApplicable(config ImageConfiguration) bool {
	if config.RuntimeType == types.Runtimes.Python {
		log.Entry(context.TODO()).Infof("Artifact %q has python runtime: specified by user in skaffold config", config.Artifact)
		return true
	}
	if hasCommonPythonEnvVars(config.Env) {
		return true
	}

	if len(config.Entrypoint) > 0 && !isEntrypointLauncher(config.Entrypoint) {
		return isLaunchingPython(config.Entrypoint)
	}
	return isLaunchingPython(config.Arguments)
}

// Apply configures a container definition for Python with ptvsd/debugpy/pydevd.
// Returns a simple map describing the debug configuration details.
func (t pythonTransformer) Apply(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator, overrideProtocols []string) (types.ContainerDebugConfiguration, string, error) {
	container := adapter.GetContainer()
	log.Entry(context.TODO()).Infof("Configuring %q for python debugging", container.Name)

	// try to find existing `-mptvsd` or `-mdebugpy` command
	if spec := retrievePythonDebugSpec(config); spec != nil {
		protocol := spec.protocol()
		container.Ports = exposePort(container.Ports, protocol, spec.port)
		return types.ContainerDebugConfiguration{
			Runtime: "python",
			Ports:   map[string]uint32{protocol: uint32(spec.port)},
		}, "", nil
	}

	spec := createPythonDebugSpec(overrideProtocols, portAlloc)

	switch {
	case isLaunchingPython(config.Entrypoint):
		container.Command = rewritePythonCommandLine(config.Entrypoint, *spec)

	case (len(config.Entrypoint) == 0 || isEntrypointLauncher(config.Entrypoint)) && isLaunchingPython(config.Arguments):
		container.Args = rewritePythonCommandLine(config.Arguments, *spec)

	case hasCommonPythonEnvVars(config.Env):
		container.Command = rewritePythonCommandLine(config.Entrypoint, *spec)

	default:
		return types.ContainerDebugConfiguration{}, "", fmt.Errorf("%q does not appear to invoke python", container.Name)
	}

	protocol := spec.protocol()
	container.Ports = exposePort(container.Ports, protocol, spec.port)

	return types.ContainerDebugConfiguration{
		Runtime: "python",
		Ports:   map[string]uint32{protocol: uint32(spec.port)},
	}, "python", nil
}

func retrievePythonDebugSpec(config ImageConfiguration) *pythonSpec {
	if spec := extractPythonDebugSpec(config.Entrypoint); spec != nil {
		return spec
	}
	if spec := extractPythonDebugSpec(config.Arguments); spec != nil {
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

func createPythonDebugSpec(overrideProtocols []string, portAlloc PortAllocator) *pythonSpec {
	for _, p := range overrideProtocols {
		switch p {
		case pydevdProtocol:
			return &pythonSpec{debugger: pydevd, port: portAlloc(defaultPydevdPort)}
		case dapProtocol:
			return &pythonSpec{debugger: debugpy, port: portAlloc(defaultDebugpyPort)}
		}
	}

	return &pythonSpec{debugger: debugpy, port: portAlloc(defaultDebugpyPort)}
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
				log.Entry(context.TODO()).Errorf("Invalid python ptvsd port %q: %s\n", args[i+1], err)
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
				log.Entry(context.TODO()).Errorf("Invalid port %q: %s\n", args[i+1], err)
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
	if index := stringslice.Index(args, "-m"+module); index >= 0 {
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

// rewritePythonCommandLine rewrites a python command-line to use the debug-support's launcher.
func rewritePythonCommandLine(commandLine []string, spec pythonSpec) []string {
	// Assumes that commandLine[0] is "python" or "python3" etc
	return stringslice.Insert(commandLine, 0, spec.asArguments())
}

func (spec pythonSpec) asArguments() []string {
	args := []string{"/dbg/python/launcher", "--mode", spec.launcherMode()}
	if spec.port >= 0 {
		args = append(args, "--port", strconv.FormatInt(int64(spec.port), 10))
	}
	if spec.wait {
		args = append(args, "--wait")
	}
	args = append(args, "--")
	return args
}

func (spec pythonSpec) launcherMode() string {
	switch spec.debugger {
	case pydevd:
		return "pydevd"
	case ptvsd:
		return "ptvsd"
	case debugpy:
		return "debugpy"
	}
	log.Entry(context.TODO()).Fatalf("invalid debugger type: %q", spec.debugger)
	return ""
}

func (spec pythonSpec) protocol() string {
	switch spec.debugger {
	case pydevd:
		return pydevdProtocol
	case debugpy, ptvsd:
		return dapProtocol
	default:
		log.Entry(context.TODO()).Fatalf("invalid debugger type: %q", spec.debugger)
		return dapProtocol
	}
}
