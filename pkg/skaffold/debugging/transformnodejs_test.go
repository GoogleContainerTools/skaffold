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

package debugging

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)


func TestExtractInspectArg(t *testing.T) {
	tests := []struct {
		in     string
		result *inspectSpec
	}{
		{"", nil},
		{"foo", nil},
		{"--foo", nil},
		{"-inspect", nil},
		{"-inspect=9329", nil},
		{"--inspect", &inspectSpec{port: 9229, brk: false}},
		{"--inspect=9329", &inspectSpec{port: 9329, brk: false}},
		{"--inspect=:9329", &inspectSpec{port: 9329, brk: false}},
		{"--inspect=foo:9329", &inspectSpec{host: "foo", port: 9329, brk: false}},
		{"--inspect-brk", &inspectSpec{port: 9229, brk: true}},
		{"--inspect-brk=9329", &inspectSpec{port: 9329, brk: true}},
		{"--inspect-brk=:9329", &inspectSpec{port: 9329, brk: true}},
		{"--inspect-brk=foo:9329", &inspectSpec{host: "foo", port: 9329, brk: true}},
	}
	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			if test.result == nil {
				testutil.CheckEqual(t, nil, test.result, extractInspectArg(test.in))
			} else {
				testutil.CheckEqual(t, cmp.Options{cmp.AllowUnexported(inspectSpec{})}, *test.result, *extractInspectArg(test.in))
			}
		})
	}
}

func TestNodeTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      imageConfiguration
		result      bool
	}{
		{
			description: "NODE_VERSION",
			source:      imageConfiguration{env: map[string]string{"NODE_VERSION": "10"}},
			result:      true,
		},
		{
			description: "entrypoint node",
			source:      imageConfiguration{entrypoint: []string{"node", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/node",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/node", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args node",
			source:      imageConfiguration{arguments: []string{"node", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/node",
			source:      imageConfiguration{arguments: []string{"/usr/bin/node", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint nodemon",
			source:      imageConfiguration{entrypoint: []string{"nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/nodemon",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args nodemon",
			source:      imageConfiguration{arguments: []string{"nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/nodemon",
			source:      imageConfiguration{arguments: []string{"/usr/bin/nodemon", "init.js"}},
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
			result := nodeTransformer{}.IsApplicable(test.source)
			testutil.CheckDeepEqual(t, test.result, result)
		})
	}
}

func TestNodeTransformerApply(t *testing.T) {
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
			result:        v1.Container{},
		},
		{
			description:   "basic",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=9229"},
				Ports:   []v1.ContainerPort{v1.ContainerPort{Name: "devtools", ContainerPort: 9229}},
			},
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{v1.ContainerPort{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: imageConfiguration{entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=9229"},
				Ports:   []v1.ContainerPort{v1.ContainerPort{Name: "http-server", ContainerPort: 8080}, v1.ContainerPort{Name: "devtools", ContainerPort: 9229}},
			},
		},
		{
			description: "command not entrypoint",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{arguments: []string{"node"}},
			result: v1.Container{
				Args: []string{"node", "--inspect=9229"},
				Ports:   []v1.ContainerPort{v1.ContainerPort{Name: "devtools", ContainerPort: 9229}},
			},
		},
	}
	var identity portAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			nodeTransformer{}.Apply(&test.containerSpec, test.configuration, identity)
			testutil.CheckDeepEqual(t, test.result, test.containerSpec)
		})
	}
}
