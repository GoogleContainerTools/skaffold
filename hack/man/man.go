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

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd"
)

func main() {
	printMan(os.Stdout, os.Stderr)
}

func printMan(stdout, stderr io.Writer) {
	command := cmd.NewSkaffoldCommand(stdout, stderr)
	printCommand(stdout, command)
}

func printCommand(out io.Writer, command *cobra.Command) {
	if command.Hidden {
		return
	}

	command.DisableFlagsInUseLine = true

	fmt.Fprintf(out, "\n### %s\n", command.CommandPath())
	fmt.Fprintf(out, "\n%s\n", command.Short)
	fmt.Fprintf(out, "\n```\n%s\n\n```\n", command.UsageString())

	if command.HasLocalFlags() {
		fmt.Fprint(out, "Env vars:\n\n")

		command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			if !flag.Hidden {
				fmt.Fprintf(out, "* `%s` (same as `--%s`)\n", cmd.FlagToEnvVarName(flag), flag.Name)
			}
		})
	}

	for _, subCommand := range command.Commands() {
		printCommand(out, subCommand)
	}
}
