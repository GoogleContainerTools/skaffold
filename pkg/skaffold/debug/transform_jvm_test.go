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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestJdwpTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      ImageConfiguration
		launcher    string
		result      bool
	}{
		{
			description: "user specified",
			source:      ImageConfiguration{RuntimeType: types.Runtimes.JVM},
			result:      true,
		},
		{
			description: "JAVA_TOOL_OPTIONS",
			source:      ImageConfiguration{Env: map[string]string{"JAVA_TOOL_OPTIONS": "-agent:jdwp"}},
			result:      true,
		},
		{
			description: "JAVA_VERSION",
			source:      ImageConfiguration{Env: map[string]string{"JAVA_VERSION": "8"}},
			result:      true,
		},
		{
			description: "entrypoint java",
			source:      ImageConfiguration{Entrypoint: []string{"java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/java",
			source:      ImageConfiguration{Entrypoint: []string{"/usr/bin/java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "no entrypoint, args java",
			source:      ImageConfiguration{Arguments: []string{"java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/java",
			source:      ImageConfiguration{Arguments: []string{"/usr/bin/java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "launcher entrypoint",
			source:      ImageConfiguration{Entrypoint: []string{"launcher"}, Arguments: []string{"/usr/bin/java", "-jar", "foo.jar"}},
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
			result := jdwpTransformer{}.IsApplicable(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestParseJdwpSpec(t *testing.T) {
	tests := []struct {
		in     string
		result jdwpSpec
	}{
		{"", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}},
		{"transport=foo", jdwpSpec{transport: "foo", quiet: false, suspend: true, server: false, host: "", port: 0}},
		{"quiet=n", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}},
		{"quiet=y", jdwpSpec{transport: "dt_socket", quiet: true, suspend: true, server: false, host: "", port: 0}},
		{"server=n", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}},
		{"server=y", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: true, host: "", port: 0}},
		{"suspend=y", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}},
		{"suspend=n", jdwpSpec{transport: "dt_socket", quiet: false, suspend: false, server: false, host: "", port: 0}},
		{"address=5005", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 5005}},
		{"address=:5005", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 5005}},
		{"address=localhost:5005", jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "localhost", port: 5005}},
		{"address=localhost:5005,quiet=y,server=y,suspend=n", jdwpSpec{transport: "dt_socket", quiet: true, suspend: false, server: true, host: "localhost", port: 5005}},
	}
	for _, test := range tests {
		testutil.Run(t, test.in, func(t *testutil.T) {
			t.CheckDeepEqual(test.result, *parseJdwpSpec(test.in), cmp.AllowUnexported(jdwpSpec{}))
			t.CheckDeepEqual(test.result, *extractJdwpArg("-agentlib:jdwp=" + test.in), cmp.AllowUnexported(jdwpSpec{}))
			t.CheckDeepEqual(test.result, *extractJdwpArg("-Xrunjdwp:" + test.in), cmp.AllowUnexported(jdwpSpec{}))
		})
	}
}

func TestJdwpSpecString(t *testing.T) {
	tests := []struct {
		in     jdwpSpec
		result string
	}{
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}, "transport=dt_socket"},
		{jdwpSpec{transport: "foo", quiet: false, suspend: true, server: false, host: "", port: 0}, "transport=foo"},
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}, "transport=dt_socket"},
		{jdwpSpec{transport: "dt_socket", quiet: true, suspend: true, server: false, host: "", port: 0}, "transport=dt_socket,quiet=y"},
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}, "transport=dt_socket"},
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: true, host: "", port: 0}, "transport=dt_socket,server=y"},
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 0}, "transport=dt_socket"},
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: false, server: false, host: "", port: 0}, "transport=dt_socket,suspend=n"},
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "", port: 5005}, "transport=dt_socket,address=5005"},
		{jdwpSpec{transport: "dt_socket", quiet: false, suspend: true, server: false, host: "localhost", port: 5005}, "transport=dt_socket,address=localhost:5005"},
	}
	for _, test := range tests {
		testutil.Run(t, test.result, func(t *testutil.T) {
			t.CheckDeepEqual(test.result, test.in.String())
		})
	}
}
