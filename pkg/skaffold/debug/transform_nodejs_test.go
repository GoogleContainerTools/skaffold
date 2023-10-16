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
		testutil.Run(t, test.in, func(t *testutil.T) {
			if test.result == nil {
				t.CheckDeepEqual(test.result, extractInspectArg(test.in))
			} else {
				t.CheckDeepEqual(*test.result, *extractInspectArg(test.in), cmp.AllowUnexported(inspectSpec{}))
			}
		})
	}
}

func TestNodeTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      ImageConfiguration
		launcher    string
		result      bool
	}{

		{
			description: "user specified",
			source:      ImageConfiguration{RuntimeType: types.Runtimes.NodeJS},
			result:      true,
		},
		{
			description: "NODE_VERSION",
			source:      ImageConfiguration{Env: map[string]string{"NODE_VERSION": "10"}},
			result:      true,
		},
		{
			description: "NODEJS_VERSION",
			source:      ImageConfiguration{Env: map[string]string{"NODEJS_VERSION": "12"}},
			result:      true,
		},
		{
			description: "NODE_ENV",
			source:      ImageConfiguration{Env: map[string]string{"NODE_ENV": "production"}},
			result:      true,
		},
		{
			description: "entrypoint node",
			source:      ImageConfiguration{Entrypoint: []string{"node", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/node",
			source:      ImageConfiguration{Entrypoint: []string{"/usr/bin/node", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args node",
			source:      ImageConfiguration{Arguments: []string{"node", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/node",
			source:      ImageConfiguration{Arguments: []string{"/usr/bin/node", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint nodemon",
			source:      ImageConfiguration{Entrypoint: []string{"nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/nodemon",
			source:      ImageConfiguration{Entrypoint: []string{"/usr/bin/nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args nodemon",
			source:      ImageConfiguration{Arguments: []string{"nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/nodemon",
			source:      ImageConfiguration{Arguments: []string{"/usr/bin/nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint npm",
			source:      ImageConfiguration{Entrypoint: []string{"npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/npm",
			source:      ImageConfiguration{Entrypoint: []string{"/usr/bin/npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "no entrypoint, args npm",
			source:      ImageConfiguration{Arguments: []string{"npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments npm",
			source:      ImageConfiguration{Arguments: []string{"npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/npm",
			source:      ImageConfiguration{Arguments: []string{"/usr/bin/npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "entrypoint /bin/sh",
			source:      ImageConfiguration{Entrypoint: []string{"/bin/sh"}},
			result:      false,
		},
		{
			description: "entrypoint launcher", // `node` image docker-entrypoint.sh"
			source:      ImageConfiguration{Entrypoint: []string{"docker-entrypoint.sh"}, Arguments: []string{"npm", "run", "dev"}},
			launcher:    "docker-entrypoint.sh",
			result:      true,
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
			result := nodeTransformer{}.IsApplicable(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestRewriteNodeCommandLine(t *testing.T) {
	tests := []struct {
		in     []string
		result []string
	}{
		{[]string{"node", "index.js"}, []string{"node", "--inspect=9226", "index.js"}},
		{[]string{"node"}, []string{"node", "--inspect=9226"}},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			result := rewriteNodeCommandLine(test.in, inspectSpec{port: 9226})

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestRewriteNpmCommandLine(t *testing.T) {
	tests := []struct {
		in     []string
		result []string
	}{
		{[]string{"npm", "run", "server"}, []string{"npm", "run", "server", "--node-options=--inspect=9226"}},
		{[]string{"npm", "run", "server", "--", "option"}, []string{"npm", "run", "server", "--node-options=--inspect=9226", "--", "option"}},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			result := rewriteNpmCommandLine(test.in, inspectSpec{port: 9226})

			t.CheckDeepEqual(test.result, result)
		})
	}
}
