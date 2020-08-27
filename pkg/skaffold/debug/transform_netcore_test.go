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
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNetcoreTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      imageConfiguration
		launcher    string
		result      bool
	}{
		{
			description: "ASPNETCORE_URLS",
			source:      imageConfiguration{env: map[string]string{"ASPNETCORE_URLS": "http://+:80"}},
			result:      true,
		},
		{
			description: "DOTNET_RUNNING_IN_CONTAINER",
			source:      imageConfiguration{env: map[string]string{"DOTNET_RUNNING_IN_CONTAINER": "true"}},
			result:      true,
		},
		{
			description: "DOTNET_SYSTEM_GLOBALIZATION_INVARIANT",
			source:      imageConfiguration{env: map[string]string{"DOTNET_SYSTEM_GLOBALIZATION_INVARIANT": "true"}},
			result:      true,
		},
		{
			description: "entrypoint with dotnet",
			source:      imageConfiguration{entrypoint: []string{"dotnet", "myapp.dll"}},
			result:      true,
		},
		{
			description: "entrypoint /bin/sh",
			source:      imageConfiguration{entrypoint: []string{"/bin/sh"}},
			result:      false,
		},
		{
			description: "launcher entrypoint exec",
			source:      imageConfiguration{entrypoint: []string{"launcher"}, arguments: []string{"exec", "dotnet", "myapp.dll"}},
			launcher:    "launcher",
			result:      true,
		},
		{
			description: "launcher entrypoint and random dotnet string",
			source:      imageConfiguration{entrypoint: []string{"launcher"}, arguments: []string{"echo", "dotnet"}},
			launcher:    "launcher",
			result:      false,
		},
		{
			description: "nothing",
			source:      imageConfiguration{},
			result:      false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&entrypointLaunchers, []string{test.launcher})
			result := netcoreTransformer{}.IsApplicable(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestNetcoreTransformerApply(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration imageConfiguration
		shouldErr     bool
		result        v1.Container
		debugConfig   ContainerDebugConfiguration
		image         string
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{},

			debugConfig: ContainerDebugConfiguration{Runtime: "netcore"},
			image:       "netcore",
			shouldErr:   false,
		},
		{
			description:   "basic",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{entrypoint: []string{"dotnet", "myapp.dll"}},

			result:      v1.Container{},
			debugConfig: ContainerDebugConfiguration{Runtime: "netcore"},
			image:       "netcore",
			shouldErr:   false,
		},
	}
	var identity portAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			config, image, err := netcoreTransformer{}.Apply(&test.containerSpec, test.configuration, identity)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.result, test.containerSpec)
			t.CheckDeepEqual(test.debugConfig, config)
			t.CheckDeepEqual(test.image, image)
		})
	}
}
