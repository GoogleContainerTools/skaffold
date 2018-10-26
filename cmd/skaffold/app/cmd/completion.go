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

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	// Only bash is supported for now. However, having args after
	// "completion" will help when supporting multiple shells
	Use: "completion bash",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("requires 1 arg, found %d", len(args))
		}
		return cobra.OnlyValidArgs(cmd, args)
	},
	ValidArgs: []string{"bash"},
	Short:     "Output command completion script for the bash shell",
	Long: `To enable command completion run

eval "$(skaffold completion bash)"

To configure bash shell completion for all your sessions, add the following to your
~/.bashrc or ~/.bash_profile:

eval "$(skaffold completion bash)"`,
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenBashCompletion(os.Stdout)
	},
}

// NewCmdCompletion returns the cobra command that outputs shell completion code
func NewCmdCompletion(out io.Writer) *cobra.Command {
	return completionCmd
}
