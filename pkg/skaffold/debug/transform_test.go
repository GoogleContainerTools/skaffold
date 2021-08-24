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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestExposePort(t *testing.T) {
	tests := []struct {
		description string
		in          []types.ContainerPort
		expected    []types.ContainerPort
	}{
		{"no ports", []types.ContainerPort{}, []types.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"existing port", []types.ContainerPort{{Name: "name", ContainerPort: 5555}}, []types.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"add new port", []types.ContainerPort{{Name: "foo", ContainerPort: 4444}}, []types.ContainerPort{{Name: "foo", ContainerPort: 4444}, {Name: "name", ContainerPort: 5555}}},
		{"clashing port name", []types.ContainerPort{{Name: "name", ContainerPort: 4444}}, []types.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"clashing port value", []types.ContainerPort{{Name: "foo", ContainerPort: 5555}}, []types.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"clashing port name and value", []types.ContainerPort{{ContainerPort: 5555}, {Name: "name", ContainerPort: 4444}}, []types.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"clashing port name and value", []types.ContainerPort{{Name: "name", ContainerPort: 4444}, {ContainerPort: 5555}}, []types.ContainerPort{{Name: "name", ContainerPort: 5555}}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// ports := k8sPortsToContainerPorts(test.in)
			result := exposePort(test.in, "name", 5555)
			// result := containerPortsToK8sPorts(ports)
			t.CheckDeepEqual(test.expected, result)
			t.CheckDeepEqual([]types.ContainerPort{{Name: "name", ContainerPort: 5555}}, filter(result, func(p types.ContainerPort) bool { return p.Name == "name" }))
			t.CheckDeepEqual([]types.ContainerPort{{Name: "name", ContainerPort: 5555}}, filter(result, func(p types.ContainerPort) bool { return p.ContainerPort == 5555 }))
		})
	}
}

func filter(ports []types.ContainerPort, predicate func(types.ContainerPort) bool) []types.ContainerPort {
	var selected []types.ContainerPort
	for _, p := range ports {
		if predicate(p) {
			selected = append(selected, p)
		}
	}
	return selected
}

func TestSetEnvVar(t *testing.T) {
	tests := []struct {
		description string
		in          types.ContainerEnv
		expected    types.ContainerEnv
	}{
		{
			description: "no entry",
			in:          types.ContainerEnv{Env: map[string]string{}},
			expected:    types.ContainerEnv{Order: []string{"name"}, Env: map[string]string{"name": "new-text"}},
		},
		{
			description: "add new entry",
			in:          types.ContainerEnv{Order: []string{"foo"}, Env: map[string]string{"foo": "bar"}},
			expected:    types.ContainerEnv{Order: []string{"foo", "name"}, Env: map[string]string{"foo": "bar", "name": "new-text"}},
		},
		{
			description: "add new entry",
			in:          types.ContainerEnv{Order: []string{"name"}, Env: map[string]string{"name": "value"}},
			expected:    types.ContainerEnv{Order: []string{"name"}, Env: map[string]string{"name": "new-text"}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := setEnvVar(test.in, "name", "new-text")
			t.CheckDeepEqual(test.expected, result)
		})
	}
}

func TestShJoin(t *testing.T) {
	tests := []struct {
		in     []string
		result string
	}{
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a b"}, `"a b"`},
		{[]string{`a"b`}, `"a\"b"`},
		{[]string{`a"b`}, `"a\"b"`},
		{[]string{"a", `a"b`, "b c"}, `a "a\"b" "b c"`},
		{[]string{"a", "b'c'd"}, `a "b'c'd"`},
		{[]string{"a", "b()"}, `a "b()"`},
		{[]string{"a", "b[]"}, `a "b[]"`},
		{[]string{"a", "b{}"}, `a "b{}"`},
		{[]string{"a", "$PORT", "${PORT}", "a ${PORT} and $PORT"}, `a $PORT "${PORT}" "a ${PORT} and $PORT"`},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			result := shJoin(test.in)
			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestIsEntrypointLauncher(t *testing.T) {
	tests := []struct {
		description string
		entrypoint  []string
		expected    bool
	}{
		{"nil", nil, false},
		{"expected case", []string{"launcher"}, true},
		{"launchers do not take args", []string{"launcher", "bar"}, false},
		{"non-launcher", []string{"/bin/sh"}, false},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&entrypointLaunchers, []string{"launcher"})
			result := isEntrypointLauncher(test.entrypoint)
			t.CheckDeepEqual(test.expected, result)
		})
	}
}

func TestUpdateForShDashC(t *testing.T) {
	// This test uses a transformer that reverses the entrypoint.  As a result:
	//  - any "/bin/sh -c script" style command-line should see only the script portion reversed
	//  - any non-"/bin/sh -c" command-line should have its entrypoint reversed
	tests := []struct {
		description string
		input       ImageConfiguration
		unwrapped   ImageConfiguration
		expected    types.ExecutableContainer
	}{
		{description: "empty"},
		{
			description: "no unwrapping: entrypoint ['a', 'b']",
			input:       ImageConfiguration{Entrypoint: []string{"a", "b"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"a", "b"}},
			expected:    types.ExecutableContainer{Command: []string{"b", "a"}},
		},
		{
			description: "no unwrapping: args ['d', 'e', 'f']",
			input:       ImageConfiguration{Arguments: []string{"d", "e", "f"}},
			unwrapped:   ImageConfiguration{Arguments: []string{"d", "e", "f"}},
		},
		{
			description: "no unwrapping: entrypoint ['a', 'b'], args [d]",
			input:       ImageConfiguration{Entrypoint: []string{"a", "b"}, Arguments: []string{"d"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"a", "b"}, Arguments: []string{"d"}},
			expected:    types.ExecutableContainer{Command: []string{"b", "a"}},
		},
		{
			description: "no unwrapping: entrypoint ['/bin/sh', '-x'] (only `-c`)",
			input:       ImageConfiguration{Entrypoint: []string{"/bin/sh", "-x"}, Arguments: []string{"d"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"/bin/sh", "-x"}, Arguments: []string{"d"}},
			expected:    types.ExecutableContainer{Command: []string{"-x", "/bin/sh"}},
		},
		{
			description: "no unwrapping: entrypoint ['sh', '-c', 'foo'] (not /bin/sh)",
			input:       ImageConfiguration{Entrypoint: []string{"sh", "-c"}, Arguments: []string{"d"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"sh", "-c"}, Arguments: []string{"d"}},
			expected:    types.ExecutableContainer{Command: []string{"-c", "sh"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/sh', '-c', 'cmd']",
			input:       ImageConfiguration{Entrypoint: []string{"/bin/sh", "-c", "d e f"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"d", "e", "f"}},
			expected:    types.ExecutableContainer{Command: []string{"/bin/sh", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/sh', '-c'], args ['d e f']",
			input:       ImageConfiguration{Entrypoint: []string{"/bin/sh", "-c"}, Arguments: []string{"d e f"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"d", "e", "f"}},
			expected:    types.ExecutableContainer{Args: []string{"f e d"}},
		},
		{
			description: "unwwrapped: args ['/bin/sh', '-c', 'd e f']",
			input:       ImageConfiguration{Arguments: []string{"/bin/sh", "-c", "d e f"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"d", "e", "f"}},
			expected:    types.ExecutableContainer{Args: []string{"/bin/sh", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/bash', '-c', 'd e f']",
			input:       ImageConfiguration{Entrypoint: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"d", "e", "f"}},
			expected:    types.ExecutableContainer{Command: []string{"/bin/bash", "-c", "f e d"}},
		},
		{
			description: "entrypoint ['/bin/bash','-c'], args ['d e f']",
			input:       ImageConfiguration{Entrypoint: []string{"/bin/bash", "-c"}, Arguments: []string{"d e f"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"d", "e", "f"}},
			expected:    types.ExecutableContainer{Args: []string{"f e d"}},
		},
		{
			description: "unwwrapped: args ['/bin/bash','-c','d e f']",
			input:       ImageConfiguration{Arguments: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"d", "e", "f"}},
			expected:    types.ExecutableContainer{Args: []string{"/bin/bash", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint-launcher and args ['/bin/sh','-c','d e f']",
			input:       ImageConfiguration{Entrypoint: []string{"launcher"}, Arguments: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   ImageConfiguration{Entrypoint: []string{"d", "e", "f"}},
			expected:    types.ExecutableContainer{Args: []string{"/bin/bash", "-c", "f e d"}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&entrypointLaunchers, []string{"launcher"})

			container := types.ExecutableContainer{}
			adapter := &testAdapter{
				executable: &container,
			}
			// The transformer reverses the unwrapped entrypoint which should be reflected into the container.Entrypoint
			updateForShDashC(adapter, test.input,
				func(a types.ContainerAdapter, result ImageConfiguration) (types.ContainerDebugConfiguration, string, error) {
					t.CheckDeepEqual(test.unwrapped, result, cmp.AllowUnexported(ImageConfiguration{}))
					if len(result.Entrypoint) > 0 {
						c := adapter.GetContainer()
						c.Command = make([]string, len(result.Entrypoint))
						for i, s := range result.Entrypoint {
							c.Command[len(result.Entrypoint)-i-1] = s
						}
					}
					return types.ContainerDebugConfiguration{}, "image", nil
				})
			t.CheckDeepEqual(test.expected, container)
		})
	}
}
