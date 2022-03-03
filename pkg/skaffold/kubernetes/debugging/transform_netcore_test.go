/*
Copyright 2021 The Skaffold Authors

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

package debugging

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging/adapter"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNetcoreTransformerApply(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration debug.ImageConfiguration
		shouldErr     bool
		result        v1.Container
		debugConfig   types.ContainerDebugConfiguration
		image         string
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{},

			debugConfig: types.ContainerDebugConfiguration{Runtime: "netcore"},
			image:       "netcore",
			shouldErr:   false,
		},
		{
			description:   "basic",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"dotnet", "myapp.dll"}},

			result:      v1.Container{},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "netcore"},
			image:       "netcore",
			shouldErr:   false,
		},
	}
	var identity debug.PortAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			adapter := adapter.NewAdapter(&test.containerSpec)
			config, image, err := debug.NewNetcoreTransformer().Apply(adapter, test.configuration, identity, nil)
			adapter.Apply()

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.result, test.containerSpec)
			t.CheckDeepEqual(test.debugConfig, config)
			t.CheckDeepEqual(test.image, image)
		})
	}
}
