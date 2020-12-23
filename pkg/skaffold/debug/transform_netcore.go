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
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type netcoreTransformer struct{}

func init() {
	containerTransforms = append(containerTransforms, netcoreTransformer{})
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

func (t netcoreTransformer) IsApplicable(config imageConfiguration) bool {
	// Some official base images (eg: dotnet/core/runtime-deps) contain the following env vars
	for _, v := range []string{"ASPNETCORE_URLS", "DOTNET_RUNNING_IN_CONTAINER", "DOTNET_SYSTEM_GLOBALIZATION_INVARIANT"} {
		if _, found := config.env[v]; found {
			return true
		}
	}

	if len(config.entrypoint) > 0 && !isEntrypointLauncher(config.entrypoint) {
		return isLaunchingNetcore(config.entrypoint)
	}

	if len(config.arguments) > 0 {
		return isLaunchingNetcore(config.arguments)
	}

	return false
}

// Apply configures a container definition for vsdbg.
// Returns a simple map describing the debug configuration details.
func (t netcoreTransformer) Apply(container *v1.Container, config imageConfiguration, portAlloc portAllocator) (ContainerDebugConfiguration, string, error) {
	logrus.Infof("Configuring %q for netcore debugging", container.Name)

	return ContainerDebugConfiguration{
		Runtime: "netcore",
	}, "netcore", nil
}
