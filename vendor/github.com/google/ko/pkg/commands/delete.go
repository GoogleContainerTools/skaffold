// Copyright 2018 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"errors"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// runCmd is suitable for use with cobra.Command's Run field.
type runCmd func(*cobra.Command, []string) error

// passthru returns a runCmd that simply passes our CLI arguments
// through to a binary named command.
func passthru(command string) runCmd {
	return func(cmd *cobra.Command, _ []string) error {
		if !isKubectlAvailable() {
			return errors.New("error: kubectl is not available. kubectl must be installed to use ko delete")
		}
		ctx := cmd.Context()

		// Start building a command line invocation by passing
		// through our arguments to command's CLI.
		//nolint:gosec // We actively want to pass arguments through, so this is fine.
		ecmd := exec.CommandContext(ctx, command, os.Args[1:]...)

		// Pass through our environment
		ecmd.Env = os.Environ()
		// Pass through our stdfoo
		ecmd.Stderr = os.Stderr
		ecmd.Stdout = os.Stdout
		ecmd.Stdin = os.Stdin

		// Run it.
		return ecmd.Run()
	}
}

// addDelete augments our CLI surface with publish.
func addDelete(topLevel *cobra.Command) {
	topLevel.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: `See "kubectl help delete" for detailed usage.`,
		RunE:  passthru("kubectl"),
		// We ignore unknown flags to avoid importing everything Go exposes
		// from our commands.
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	})
}
