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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type dlvTransformer struct{}

func init() {
	containerTransforms = append(containerTransforms, dlvTransformer{})
}

const (
	// dlv defaults to port 56268
	defaultDlvPort    = 56268
	defaultAPIVersion = 2
)

// dlvSpec captures the useful delve runtime options
type dlvSpec struct {
	mode       string
	host       string
	port       uint16
	headless   bool
	log        bool
	apiVersion int
}

func newDlvSpec(port uint16) dlvSpec {
	return dlvSpec{mode: "exec", host: "localhost", port: port, apiVersion: defaultAPIVersion, headless: true}
}

// isLaunchingDlv determines if the arguments seems to be invoking Delve
func isLaunchingDlv(args []string) bool {
	return len(args) > 0 && (args[0] == "dlv" || strings.HasSuffix(args[0], "/dlv"))
}

func (t dlvTransformer) IsApplicable(config imageConfiguration) bool {
	for _, name := range []string{"GODEBUG", "GOGC", "GOMAXPROCS", "GOTRACEBACK"} {
		if _, found := config.env[name]; found {
			return true
		}
	}
	if len(config.entrypoint) > 0 {
		return isLaunchingDlv(config.entrypoint)
	}
	if len(config.arguments) > 0 {
		return isLaunchingDlv(config.arguments)
	}
	return false
}

func (t dlvTransformer) RuntimeSupportImage() string {
	return "go"
}

// Apply configures a container definition for Go with Delve.
// Returns a simple map describing the debug configuration details.
func (t dlvTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) map[string]interface{} {
	logrus.Infof("Configuring %q for Go/Delve debugging", container.Name)

	// try to find existing `dlv` command
	spec := retrieveDlvSpec(config)
	// todo: find existing containerPort "dlv" and use port. But what if it conflicts with command-line spec?

	if spec == nil {
		newSpec := newDlvSpec(uint16(portAlloc(defaultDlvPort)))
		spec = &newSpec
		switch {
		case len(config.entrypoint) > 0:
			container.Command = rewriteDlvCommandLine(config.entrypoint, *spec)

		case len(config.entrypoint) == 0 && len(config.arguments) > 0:
			container.Args = rewriteDlvCommandLine(config.arguments, *spec)

		default:
			logrus.Warnf("Skipping %q as does not appear to be Go-based", container.Name)
			return nil
		}
	}

	dlvPort := v1.ContainerPort{
		Name:          "dlv",
		ContainerPort: int32(spec.port),
	}
	container.Ports = append(container.Ports, dlvPort)

	return map[string]interface{}{
		"runtime": "go",
		"dlv":     spec.port,
	}
}

func retrieveDlvSpec(config imageConfiguration) *dlvSpec {
	if spec := extractDlvSpec(config.entrypoint); spec != nil {
		return spec
	}
	if spec := extractDlvSpec(config.arguments); spec != nil {
		return spec
	}
	return nil
}

func extractDlvSpec(args []string) *dlvSpec {
	if !isLaunchingDlv(args) {
		return nil
	}
	spec := dlvSpec{apiVersion: 2, log: false, headless: false}
arguments:
	for _, arg := range args {
		switch {
		case arg == "--":
			break arguments
		case arg == "debug" || arg == "test" || arg == "exec":
			spec.mode = arg
		case arg == "--headless":
			spec.headless = true
		case arg == "--log":
			spec.log = true
		case strings.HasPrefix(arg, "--listen="):
			address := strings.SplitN(arg, "=", 2)[1]
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
		case strings.HasPrefix(arg, "--api-version="):
			address := strings.SplitN(arg, "=", 2)[1]
			version, _ := strconv.ParseInt(address, 10, 16)
			spec.apiVersion = int(version)
		}
	}
	return &spec
}

// rewriteDlvCommandLine rewrites a go command-line to insert a `dlv`
func rewriteDlvCommandLine(commandLine []string, spec dlvSpec) []string {
	// todo: parse off dlv commands if present?

	if len(commandLine) > 1 {
		// insert "--" after app binary to indicate end of Delve arguments
		commandLine = util.StrSliceInsert(commandLine, 1, []string{"--"})
	}
	return append(spec.asArguments(), commandLine...)
}

func (spec dlvSpec) asArguments() []string {
	args := []string{"/dbg/go/bin/dlv"}
	args = append(args, spec.mode)
	if spec.headless {
		args = append(args, "--headless")
	}
	args = append(args, "--continue", "--accept-multiclient")
	host := "localhost"
	if spec.host != "" {
		host = spec.host
	}
	if spec.port > 0 {
		args = append(args, fmt.Sprintf("--listen=%s:%d", host, spec.port))
		//args = append(args, ":", strconv.FormatInt(int64(spec.port), 10))
	} else {
		args = append(args, fmt.Sprintf("--listen=%s", host))
	}
	if spec.apiVersion > 0 {
		args = append(args, fmt.Sprintf("--api-version=%d", spec.apiVersion))
	}
	if spec.log {
		args = append(args, "--log")
	}
	return args
}
