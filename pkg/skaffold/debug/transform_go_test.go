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

func TestNewDlvSpecDefaults(t *testing.T) {
	spec := newDlvSpec(20)
	expected := dlvSpec{mode: "exec", port: 20, apiVersion: 2, headless: true, log: false}
	testutil.CheckDeepEqual(t, expected, spec, cmp.AllowUnexported(spec))
}

func TestExtractDlvArg(t *testing.T) {
	tests := []struct {
		in     []string
		result *dlvSpec
	}{
		{nil, nil},
		{[]string{"foo"}, nil},
		{[]string{"foo", "--foo"}, nil},
		{[]string{"dlv", "debug", "--headless"}, &dlvSpec{mode: "debug", headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "--headless", "exec"}, &dlvSpec{mode: "exec", headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "--headless", "exec", "--", "--listen=host:4345"}, &dlvSpec{mode: "exec", headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "test", "--headless", "--listen=host:4345"}, &dlvSpec{mode: "test", host: "host", port: 4345, headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "debug", "--headless", "--api-version=1"}, &dlvSpec{mode: "debug", headless: true, apiVersion: 1, log: false}},
		{[]string{"dlv", "debug", "--listen=host:4345", "--headless", "--api-version=2", "--log"}, &dlvSpec{mode: "debug", host: "host", port: 4345, headless: true, apiVersion: 2, log: true}},
		{[]string{"dlv", "debug", "--listen=:4345"}, &dlvSpec{mode: "debug", port: 4345, apiVersion: 2}},
		{[]string{"dlv", "debug", "--listen=host:"}, &dlvSpec{mode: "debug", host: "host", apiVersion: 2}},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			if test.result == nil {
				t.CheckDeepEqual(test.result, extractDlvSpec(test.in))
			} else {
				t.CheckDeepEqual(*test.result, *extractDlvSpec(test.in), cmp.AllowUnexported(dlvSpec{}))
			}
		})
	}
}

func TestDlvTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      ImageConfiguration
		launcher    string
		result      bool
	}{
		{
			description: "user specified",
			source:      ImageConfiguration{RuntimeType: types.Runtimes.Go},
			result:      true,
		},
		{
			description: "GOMAXPROCS",
			source:      ImageConfiguration{Env: map[string]string{"GOMAXPROCS": "2"}},
			result:      true,
		},
		{
			description: "GOGC",
			source:      ImageConfiguration{Env: map[string]string{"GOGC": "off"}},
			result:      true,
		},
		{
			description: "GODEBUG",
			source:      ImageConfiguration{Env: map[string]string{"GODEBUG": "efence=1"}},
			result:      true,
		},
		{
			description: "GOTRACEBACK",
			source:      ImageConfiguration{Env: map[string]string{"GOTRACEBACK": "off"}},
			result:      true,
		},
		{
			// detect images built by ko: https://github.com/google/ko#static-assets
			description: "KO_DATA_PATH",
			source:      ImageConfiguration{Env: map[string]string{"KO_DATA_PATH": "cmd/app/kodata/"}},
			result:      true,
		},
		{
			description: "entrypoint with dlv",
			source:      ImageConfiguration{Entrypoint: []string{"dlv", "exec", "--headless"}},
			result:      true,
		},
		{
			description: "launcher entrypoint",
			source:      ImageConfiguration{Entrypoint: []string{"launcher"}, Arguments: []string{"dlv", "exec", "--headless"}},
			launcher:    "launcher",
			result:      true,
		},
		{
			description: "ko author",
			source:      ImageConfiguration{Author: "github.com/google/ko"},
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
			result := dlvTransformer{}.IsApplicable(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}
