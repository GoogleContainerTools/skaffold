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

type dlvTransformer struct{}

//nolint:golint
func NewDlvTransformer() containerTransformer {
	return dlvTransformer{}
}

func init() {
	RegisterContainerTransformer(NewDlvTransformer())
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
	return dlvSpec{mode: "exec", port: port, apiVersion: defaultAPIVersion, headless: true}
}

// isLaunchingDlv determines if the arguments seems to be invoking Delve
func isLaunchingDlv(args []string) bool {
	return len(args) > 0 && (args[0] == "dlv" || strings.HasSuffix(args[0], "/dlv"))
}

func (t dlvTransformer) IsApplicable(config ImageConfiguration) bool {
	if config.RuntimeType == types.Runtimes.Go {
		log.Entry(context.TODO()).Infof("Artifact %q has Go runtime: specified by user in skaffold config", config.Artifact)
		return true
	}
	for _, name := range []string{"GODEBUG", "GOGC", "GOMAXPROCS", "GOTRACEBACK", "KO_DATA_PATH"} {
		if _, found := config.Env[name]; found {
			log.Entry(context.TODO()).Infof("Artifact %q has Go runtime: has env %q", config.Artifact, name)
			return true
		}
	}
	// Detect ko image by author, see https://github.com/google/ko/blob/v0.8.3/pkg/build/gobuild.go#L610
	if config.Author == "github.com/google/ko" {
		log.Entry(context.TODO()).Infof("Artifact %q has Go runtime: has author %q", config.Artifact, config.Author)
		return true
	}

	// FIXME: as there is currently no way to identify a buildpacks-produced image as holding a Go binary,
	// nor to cause certain environment variables to be defined in the resulting image, look at the image's
	// CNB metadata to see if any well-known Go-related buildpacks had been involved.
	knownGoBuildpackIds := []string{
		"google.go.build",           // GCP Buildpacks
		"paketo-buildpacks/go-dist", // Paketo
		"heroku/go",                 // Heroku
	}
	cnbBuildMetadata := config.Labels["io.buildpacks.build.metadata"]
	for _, id := range knownGoBuildpackIds {
		if strings.Contains(cnbBuildMetadata, id) {
			log.Entry(context.TODO()).Infof("Artifact %q has Go buildpacks %q", config.Artifact, id)
			return true
		}
	}
	if len(config.Entrypoint) > 0 && !isEntrypointLauncher(config.Entrypoint) {
		return isLaunchingDlv(config.Entrypoint)
	}
	if len(config.Arguments) > 0 {
		return isLaunchingDlv(config.Arguments)
	}
	return false
}

// Apply configures a container definition for Go with Delve.
// Returns the debug configuration details, with the "go" support image
func (t dlvTransformer) Apply(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator, overrideProtocols []string) (types.ContainerDebugConfiguration, string, error) {
	container := adapter.GetContainer()
	log.Entry(context.TODO()).Infof("Configuring %q for Go/Delve debugging", container.Name)

	// try to find existing `dlv` command
	spec := retrieveDlvSpec(config)

	if spec == nil {
		newSpec := newDlvSpec(uint16(portAlloc(defaultDlvPort)))
		spec = &newSpec
		switch {
		case len(config.Entrypoint) > 0 && !isEntrypointLauncher(config.Entrypoint):
			container.Command = rewriteDlvCommandLine(config.Entrypoint, *spec, container.Args)

		case (len(config.Entrypoint) == 0 || isEntrypointLauncher(config.Entrypoint)) && len(config.Arguments) > 0:
			container.Args = rewriteDlvCommandLine(config.Arguments, *spec, container.Args)

		default:
			return types.ContainerDebugConfiguration{}, "", fmt.Errorf("container %q has no command-line", container.Name)
		}
	}

	container.Ports = exposePort(container.Ports, "dlv", int32(spec.port))

	return types.ContainerDebugConfiguration{
		Runtime: "go",
		Ports:   map[string]uint32{"dlv": uint32(spec.port)},
	}, "go", nil
}

func retrieveDlvSpec(config ImageConfiguration) *dlvSpec {
	if spec := extractDlvSpec(config.Entrypoint); spec != nil {
		return spec
	}
	if spec := extractDlvSpec(config.Arguments); spec != nil {
		return spec
	}
	return nil
}

func extractDlvSpec(args []string) *dlvSpec {
	if !isLaunchingDlv(args) {
		return nil
	}
	// delve's defaults
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
			case 1:
				// this is actually an error: delve insists on a :port
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
func rewriteDlvCommandLine(commandLine []string, spec dlvSpec, args []string) []string {
	// todo: parse off dlv commands if present?
	if len(commandLine) > 1 || len(args) > 0 {
		// insert "--" after app binary to indicate end of Delve arguments
		commandLine = stringslice.Insert(commandLine, 1, []string{"--"})
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
	if spec.port > 0 {
		args = append(args, fmt.Sprintf("--listen=%s:%d", spec.host, spec.port))
	} else {
		args = append(args, fmt.Sprintf("--listen=%s", spec.host))
	}
	if spec.apiVersion > 0 {
		args = append(args, fmt.Sprintf("--api-version=%d", spec.apiVersion))
	}
	if spec.log {
		args = append(args, "--log")
	}
	return args
}
