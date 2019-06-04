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

package kubectl

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestCheckVersion(t *testing.T) {
	var tests = []struct {
		description string
		command     util.Command
		shouldErr   bool
		warnings    []string
	}{
		{
			description: "1.12 is valid",
			command:     testutil.FakeRunOut(t, "kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"12"}}`),
		},
		{
			description: "1.12+ is valid",
			command:     testutil.FakeRunOut(t, "kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"12+"}}`),
		},
		{
			description: "1.11 is too old",
			command:     testutil.FakeRunOut(t, "kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"11"}}`),
			shouldErr:   true,
		},
		{
			description: "invalid version",
			command:     testutil.FakeRunOut(t, "kubectl version --client -ojson", `not json`),
			shouldErr:   true,
			warnings:    []string{"unable to parse client version: invalid character 'o' in literal null (expecting 'u')"},
		},
		{
			description: "cli not found",
			command:     testutil.FakeRunOutErr(t, "kubectl version --client -ojson", ``, errors.New("not found")),
			shouldErr:   true,
			warnings:    []string{"unable to get kubectl client version: not found"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.DefaultExecCommand, test.command)

			cli := CLI{}
			err := cli.CheckVersion(context.Background())

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.warnings, fakeWarner.Warnings)
		})
	}
}
