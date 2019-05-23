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
	"io"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewCmdConfig(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "A set of commands for interacting with the Skaffold config.",
	}

	cmd.AddCommand(NewCmdSet(out))
	cmd.AddCommand(NewCmdUnset(out))
	cmd.AddCommand(NewCmdList(out))
	return cmd
}

func NewCmdSet(out io.Writer) *cobra.Command {
	return NewCmd(out, "set").
		WithDescription("Set a value in the global Skaffold config").
		WithFlags(func(f *pflag.FlagSet) {
			config.AddCommonFlags(f)
			config.AddSetUnsetFlags(f)
		}).
		ExactArgs(2, config.Set)
}

func NewCmdUnset(out io.Writer) *cobra.Command {
	return NewCmd(out, "unset").
		WithDescription("Unset a value in the global Skaffold config").
		WithFlags(func(f *pflag.FlagSet) {
			config.AddCommonFlags(f)
			config.AddSetUnsetFlags(f)
		}).
		ExactArgs(1, config.Unset)
}

func NewCmdList(out io.Writer) *cobra.Command {
	return NewCmd(out, "list").
		WithDescription("List all values set in the global Skaffold config").
		WithFlags(func(f *pflag.FlagSet) {
			config.AddCommonFlags(f)
			config.AddListFlags(f)
		}).
		NoArgs(config.List)
}
