/*
Copyright 2026 The Skaffold Authors

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

package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

// TestDeployParamsFlagsAvailable verifies that --set and --set-value-file
// are registered on every command through which deploy parameters can flow:
// render, delete (pre-existing) + deploy, dev, run, exec, verify (added for
// Cloud Deploy custom-target / verify parity).
//
// Note: the filter command also accepts --set but not --set-value-file, so
// it is intentionally excluded from this check.
func TestDeployParamsFlagsAvailable(t *testing.T) {
	cases := []struct {
		name string
		ctor func() *cobra.Command
	}{
		{name: "render", ctor: NewCmdRender},
		{name: "delete", ctor: NewCmdDelete},
		{name: "deploy", ctor: NewCmdDeploy},
		{name: "dev", ctor: NewCmdDev},
		{name: "run", ctor: NewCmdRun},
		{name: "exec", ctor: NewCmdExec},
		{name: "verify", ctor: NewCmdVerify},
	}

	for _, tc := range cases {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			t.NewTempDir().Chdir()
			t.Override(&opts, config.SkaffoldOptions{})

			cmd := tc.ctor()
			cmd.SilenceUsage = true

			t.CheckTrue(cmd.Flags().Lookup("set") != nil)
			t.CheckTrue(cmd.Flags().Lookup("set-value-file") != nil)
		})
	}
}

// TestDeployParamsFlagsBindToOpts verifies the flag values bind through to
// opts.ManifestsOverrides / opts.ManifestsValueFile on the newly added
// commands, mirroring the existing TestNewCmdDelete coverage.
func TestDeployParamsFlagsBindToOpts(t *testing.T) {
	cases := []struct {
		name string
		ctor func() *cobra.Command
	}{
		{name: "deploy", ctor: NewCmdDeploy},
		{name: "dev", ctor: NewCmdDev},
		{name: "run", ctor: NewCmdRun},
		{name: "exec", ctor: NewCmdExec},
		{name: "verify", ctor: NewCmdVerify},
	}

	for _, tc := range cases {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			t.NewTempDir().Chdir()
			t.Override(&opts, config.SkaffoldOptions{})

			cmd := tc.ctor()
			cmd.SilenceUsage = true

			err := cmd.Flags().Parse([]string{"--set", "key1=val1", "--set", "key2=val2"})
			t.CheckNoError(err)
			t.CheckDeepEqual([]string{"key1=val1", "key2=val2"}, opts.ManifestsOverrides)

			err = cmd.Flags().Parse([]string{"--set-value-file", "values.env"})
			t.CheckNoError(err)
			t.CheckDeepEqual("values.env", opts.ManifestsValueFile)
		})
	}
}
