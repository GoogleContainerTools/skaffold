/*
Copyright 2020 The Skaffold Authors

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
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

type netcoreTransformer struct{}

//nolint:golint
func NewNetcoreTransformer() containerTransformer {
	return netcoreTransformer{}
}

func init() {
	RegisterContainerTransformer(NewNetcoreTransformer())
}

// isLaunchingNetcore determines if the arguments seems to be invoking dotnet
func isLaunchingNetcore(args []string) bool {
	if len(args) < 2 {
		return false
	}

	if args[0] == "dotnet" || strings.HasSuffix(args[0], "/dotnet") {
		return true
	}

	if args[0] == "exec" && (args[1] == "dotnet" || strings.HasSuffix(args[1], "/dotnet")) {
		return true
	}

	return false
}

func (t netcoreTransformer) IsApplicable(config ImageConfiguration) bool {
	if config.RuntimeType == types.Runtimes.NetCore {
		log.Entry(context.TODO()).Infof("Artifact %q has netcore runtime: specified by user in skaffold config", config.Artifact)
		return true
	}
	// Some official base images (eg: dotnet/core/runtime-deps) contain the following env vars
	for _, v := range []string{"ASPNETCORE_URLS", "DOTNET_RUNNING_IN_CONTAINER", "DOTNET_SYSTEM_GLOBALIZATION_INVARIANT"} {
		if _, found := config.Env[v]; found {
			return true
		}
	}

	if len(config.Entrypoint) > 0 && !isEntrypointLauncher(config.Entrypoint) {
		return isLaunchingNetcore(config.Entrypoint)
	}

	if len(config.Arguments) > 0 {
		return isLaunchingNetcore(config.Arguments)
	}

	return false
}

// Apply configures a container definition for vsdbg.
// Returns a simple map describing the debug configuration details.
func (t netcoreTransformer) Apply(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator, overrideProtocols []string) (types.ContainerDebugConfiguration, string, error) {
	container := adapter.GetContainer()
	log.Entry(context.TODO()).Infof("Configuring %q for netcore debugging", container.Name)

	return types.ContainerDebugConfiguration{
		Runtime: "netcore",
	}, "netcore", nil
}
