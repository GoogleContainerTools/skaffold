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

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

const (
	longDescription = `
	Outputs shell completion for the given shell (bash or zsh)

	This depends on the bash-completion binary.  Example installation instructions:
	OS X:
		$ brew install bash-completion
		$ source $(brew --prefix)/etc/bash_completion
		$ skaffold completion bash > ~/.skaffold-completion  # for bash users
		$ skaffold completion zsh > ~/.skaffold-completion   # for zsh users
		$ source ~/.skaffold-completion
	Ubuntu:
		$ apt-get install bash-completion
		$ source /etc/bash-completion
		$ source <(skaffold completion bash) # for bash users
		$ source <(skaffold completion zsh)  # for zsh users

	Additionally, you may want to output the completion to a file and source in your .bashrc
`

	zshCompdef = "\ncompdef _skaffold skaffold\n"
)

func completion(cmd *cobra.Command, args []string) {
	switch args[0] {
	case "bash":
		rootCmd(cmd).GenBashCompletion(os.Stdout)
	case "zsh":
		runCompletionZsh(cmd, os.Stdout)
	}
}

// NewCmdCompletion returns the cobra command that outputs shell completion code
func NewCmdCompletion() *cobra.Command {
	return &cobra.Command{
		Use: "completion SHELL",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("requires 1 arg, found %d", len(args))
			}
			return cobra.OnlyValidArgs(cmd, args)
		},
		ValidArgs: []string{"bash", "zsh"},
		Short:     "Output shell completion for the given shell (bash or zsh)",
		Long:      longDescription,
		Run:       completion,
	}
}

func runCompletionZsh(cmd *cobra.Command, out io.Writer) {
	rootCmd(cmd).GenZshCompletion(out)
	io.WriteString(out, zshCompdef)
}

func rootCmd(cmd *cobra.Command) *cobra.Command {
	parent := cmd
	for parent.HasParent() {
		parent = parent.Parent()
	}
	return parent
}
