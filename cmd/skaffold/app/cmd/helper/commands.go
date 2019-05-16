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

package helper

import (
	"io"

	"github.com/spf13/cobra"
)

// ArgsCommand describes a command that takes a fixed number of arguments.
func ArgsCommand(out io.Writer, use, description string, argCount int, action func(io.Writer, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: description,
		Args:  cobra.ExactArgs(argCount),
		RunE: func(cmd *cobra.Command, args []string) error {
			return action(out, args)
		},
	}
}

// NoArgCommand describes a command that takes no argument.
func NoArgCommand(out io.Writer, use, description string, action func(io.Writer) error) *cobra.Command {
	return ArgsCommand(out, use, description, 0, func(out io.Writer, _ []string) error {
		return action(out)
	})
}
