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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestExtractDebugSpecs(t *testing.T) {
	tests := []struct {
		in     []string
		result *pythonSpec
	}{
		{nil, nil},
		{[]string{"foo"}, nil},
		{[]string{"--foo"}, nil},
		{[]string{"-mfoo"}, nil},
		{[]string{"-m", "foo"}, nil},
		// ptvsd has implicit port and host
		{[]string{"-mptvsd"}, &pythonSpec{debugger: ptvsd, port: 5678, wait: false}},
		{[]string{"-m", "ptvsd", "--port", "9329"}, &pythonSpec{debugger: ptvsd, port: 9329, wait: false}},
		{[]string{"-mptvsd", "--port", "9329", "--host", "foo"}, &pythonSpec{debugger: ptvsd, host: "foo", port: 9329, wait: false}},
		{[]string{"-mptvsd", "--wait"}, &pythonSpec{debugger: ptvsd, port: 5678, wait: true}},
		{[]string{"-m", "ptvsd", "--wait", "--port", "9329", "--host", "foo"}, &pythonSpec{debugger: ptvsd, host: "foo", port: 9329, wait: true}},
		// debugpy requires a port and either `--connect` or `--listen`
		{[]string{"-mdebugpy"}, nil}, // debugpy requires a port and `--listen`
		{[]string{"-mdebugpy", "--wait-for-client"}, nil},
		{[]string{"-m", "debugpy", "--listen", "9329"}, &pythonSpec{debugger: debugpy, port: 9329, wait: false}},
		{[]string{"-mdebugpy", "--listen", "foo:9329"}, &pythonSpec{debugger: debugpy, host: "foo", port: 9329, wait: false}},
		{[]string{"-m", "debugpy", "--wait-for-client", "--listen", "foo:9329"}, &pythonSpec{debugger: debugpy, host: "foo", port: 9329, wait: true}},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			if test.result == nil {
				t.CheckDeepEqual(test.result, extractPythonDebugSpec(test.in))
			} else {
				t.CheckDeepEqual(*test.result, *extractPythonDebugSpec(test.in), cmp.AllowUnexported(pythonSpec{debugger: ptvsd}))
			}
		})
	}
}

func TestPythonTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      ImageConfiguration
		launcher    string
		result      bool
	}{

		{
			description: "user specified",
			source:      ImageConfiguration{RuntimeType: types.Runtimes.Python},
			result:      true,
		},
		{
			description: "PYTHON_VERSION",
			source:      ImageConfiguration{Env: map[string]string{"PYTHON_VERSION": "2.7"}},
			result:      true,
		},
		{
			description: "entrypoint python",
			source:      ImageConfiguration{Entrypoint: []string{"python", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/python",
			source:      ImageConfiguration{Entrypoint: []string{"/usr/bin/python", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args python",
			source:      ImageConfiguration{Arguments: []string{"python", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/python",
			source:      ImageConfiguration{Arguments: []string{"/usr/bin/python", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint python2",
			source:      ImageConfiguration{Entrypoint: []string{"python2", "init.py"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/python2",
			source:      ImageConfiguration{Entrypoint: []string{"/usr/bin/python2", "init.py"}},
			result:      true,
		},
		{
			description: "no entrypoint, args python2",
			source:      ImageConfiguration{Arguments: []string{"python2", "init.py"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/python2",
			source:      ImageConfiguration{Arguments: []string{"/usr/bin/python2", "init.py"}},
			result:      true,
		},
		{
			description: "entrypoint launcher",
			source:      ImageConfiguration{Entrypoint: []string{"launcher"}, Arguments: []string{"python3", "app.py"}},
			launcher:    "launcher",
			result:      true,
		},
		{
			description: "entrypoint /bin/sh",
			source:      ImageConfiguration{Entrypoint: []string{"/bin/sh"}},
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
			result := pythonTransformer{}.IsApplicable(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}
