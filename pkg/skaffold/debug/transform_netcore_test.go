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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestNetcoreTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      ImageConfiguration
		launcher    string
		result      bool
	}{
		{
			description: "user specified",
			source:      ImageConfiguration{RuntimeType: types.Runtimes.NetCore},
			result:      true,
		},
		{
			description: "ASPNETCORE_URLS",
			source:      ImageConfiguration{Env: map[string]string{"ASPNETCORE_URLS": "http://+:80"}},
			result:      true,
		},
		{
			description: "DOTNET_RUNNING_IN_CONTAINER",
			source:      ImageConfiguration{Env: map[string]string{"DOTNET_RUNNING_IN_CONTAINER": "true"}},
			result:      true,
		},
		{
			description: "DOTNET_SYSTEM_GLOBALIZATION_INVARIANT",
			source:      ImageConfiguration{Env: map[string]string{"DOTNET_SYSTEM_GLOBALIZATION_INVARIANT": "true"}},
			result:      true,
		},
		{
			description: "entrypoint with dotnet",
			source:      ImageConfiguration{Entrypoint: []string{"dotnet", "myapp.dll"}},
			result:      true,
		},
		{
			description: "entrypoint /bin/sh",
			source:      ImageConfiguration{Entrypoint: []string{"/bin/sh"}},
			result:      false,
		},
		{
			description: "launcher entrypoint exec",
			source:      ImageConfiguration{Entrypoint: []string{"launcher"}, Arguments: []string{"exec", "dotnet", "myapp.dll"}},
			launcher:    "launcher",
			result:      true,
		},
		{
			description: "launcher entrypoint and random dotnet string",
			source:      ImageConfiguration{Entrypoint: []string{"launcher"}, Arguments: []string{"echo", "dotnet"}},
			launcher:    "launcher",
			result:      false,
		},
		{
			description: "nothing",
			source:      ImageConfiguration{},
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
