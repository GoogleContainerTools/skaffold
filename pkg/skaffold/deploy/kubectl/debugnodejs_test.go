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

package kubectl

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


func TestConfigureNodeJSDebugging(t *testing.T) {
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
			configureNodeJSDebugging(&test.containerSpec, test.configuration, identity)
			testutil.CheckDeepEqual(t, test.result, test.containerSpec)
		})
	}
}
