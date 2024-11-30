/*
Copyright 2024 The Skaffold Authors

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

package tofu

import (
	"context"
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestCheckVersion(t *testing.T) {
	tests := []struct {
		description     string
		commands        util.Command
		shouldErr       bool
		warnings        []string
		expectedVersion string
	}{
		{
			description:     "0.16.0-alpha2",
			commands:        testutil.CmdRunOut("tofu version --json", `{"terraform_version": "0.16.0-alpha2"}`),
			expectedVersion: "0.16.0-alpha2",
		},
		{
			description:     "cli not found",
			commands:        testutil.CmdRunOutErr("tofu version --json", ``, errors.New("not found")),
			warnings:        []string{"unable to get tofu version: not found"},
			expectedVersion: "unknown",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.DefaultExecCommand, test.commands)

			cli := CLI{}

			version := cli.Version(context.Background())
			t.CheckDeepEqual(test.expectedVersion, version)

			t.CheckErrorAndDeepEqual(test.shouldErr, nil, test.warnings, fakeWarner.Warnings)
		})
	}
}
