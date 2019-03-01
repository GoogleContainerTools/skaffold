/*
Copyright 2018 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestJdwpTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      imageConfiguration
		result      bool
	}{
		{
			description: "JAVA_TOOL_OPTIONS",
			source:      imageConfiguration{env: map[string]string{"JAVA_TOOL_OPTIONS": "-agent:jdwp"}},
			result:      true,
		},
		{
			description: "JAVA_VERSION",
			source:      imageConfiguration{env: map[string]string{"JAVA_VERSION": "8"}},
			result:      true,
		},
		{
			description: "entrypoint java",
			source:      imageConfiguration{entrypoint: []string{"java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/java",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "no entrypoint, args java",
			source:      imageConfiguration{arguments: []string{"java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/java",
			source:      imageConfiguration{arguments: []string{"/usr/bin/java", "-jar", "foo.jar"}},
			result:      true,
		},
		{
			description: "entrypoint /bin/sh",
			source:      imageConfiguration{entrypoint: []string{"/bin/sh"}},
			result:      false,
		},
		{
			description: "nothing",
			source:      imageConfiguration{},
			result:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := jdwpTransformer{}.IsApplicable(test.source)
			testutil.CheckDeepEqual(t, test.result, result)
		})
	}
}

func TestJdwpTransformerApply(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration imageConfiguration
		result        v1.Container
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{v1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{v1.ContainerPort{Name: "jdwp", ContainerPort: 5005}},
			},
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{v1.ContainerPort{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: imageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{v1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{v1.ContainerPort{Name: "http-server", ContainerPort: 8080}, v1.ContainerPort{Name: "jdwp", ContainerPort: 5005}},
			},
		},
		{
			description: "existing jdwp spec",
			containerSpec: v1.Container{
				Env:   []v1.EnvVar{v1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=8000,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{v1.ContainerPort{ContainerPort: 5005}},
			},
			configuration: imageConfiguration{env: map[string]string{"JAVA_TOOL_OPTIONS":"-agentlib:jdwp=transport=dt_socket,server=y,address=8000,suspend=n,quiet=y"}},
			result: v1.Container{
				Env:   []v1.EnvVar{v1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=8000,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{v1.ContainerPort{ContainerPort: 5005}, v1.ContainerPort{Name: "jdwp", ContainerPort: 8000}},
			},
		},
	}
	var identity portAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			jdwpTransformer{}.Apply(&test.containerSpec, test.configuration, identity)
			testutil.CheckDeepEqual(t, test.result, test.containerSpec)
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
		t.Run(test.in, func(t *testing.T) {
			testutil.CheckEqual(t, cmp.Options{cmp.AllowUnexported(jdwpSpec{})}, test.result, *parseJdwpSpec(test.in))
			testutil.CheckEqual(t, cmp.Options{cmp.AllowUnexported(jdwpSpec{})}, test.result, *extractJdwpArg("-agentlib:jdwp=" + test.in))
			testutil.CheckEqual(t, cmp.Options{cmp.AllowUnexported(jdwpSpec{})}, test.result, *extractJdwpArg("-Xrunjdwp:" + test.in))
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
		t.Run(test.result, func(t *testing.T) {
			testutil.CheckDeepEqual(t, test.result, test.in.String())
		})
	}
}
