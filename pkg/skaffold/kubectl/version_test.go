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
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
			description:     "1.12 is valid",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"12"}}`),
			expectedVersion: "1.12",
		},
		{
			description:     "1.12+ is valid",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"12+"}}`),
			expectedVersion: "1.12+",
		},
		{
			description:     "1.13 is valid",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"13"}}`),
			expectedVersion: "1.13",
		},
		{
			description:     "2.11 is valid",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `{"clientVersion":{"major":"2","minor":"11"}}`),
			expectedVersion: "2.11",
		},
		{
			description:     "1.11 is too old",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"11"}}`),
			shouldErr:       true,
			expectedVersion: "1.11",
		},
		{
			description:     "invalid version",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `not json`),
			shouldErr:       true,
			warnings:        []string{"unable to parse client version: invalid character 'o' in literal null (expecting 'u')"},
			expectedVersion: "unknown",
		},
		{
			description:     "invalid minor",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `{"clientVersion":{"major":"1","minor":"X"}}`),
			shouldErr:       true,
			expectedVersion: "1.X",
		},
		{
			description:     "invalid major",
			commands:        testutil.CmdRunOut("kubectl version --client -ojson", `{"clientVersion":{"major":"X","minor":"1"}}`),
			shouldErr:       true,
			expectedVersion: "X.1",
		},
		{
			description:     "cli not found",
			commands:        testutil.CmdRunOutErr("kubectl version --client -ojson", ``, errors.New("not found")),
			shouldErr:       true,
			warnings:        []string{"unable to get kubectl client version: not found"},
			expectedVersion: "unknown",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.DefaultExecCommand, test.commands)

			cli := CLI{}

			version := cli.Version(context.Background()).String()
			t.CheckDeepEqual(test.expectedVersion, version)

			err := cli.CheckVersion(context.Background())
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.warnings, fakeWarner.Warnings)
		})
	}
}
