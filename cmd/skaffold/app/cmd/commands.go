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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Builder is used to build cobra commands.
type Builder interface {
	WithDescription(description string) Builder
	WithLongDescription(long string) Builder
	WithFlags(adder func(*pflag.FlagSet)) Builder
	WithCommonFlags() Builder
	ExactArgs(argCount int, action func(io.Writer, []string) error) *cobra.Command
	NoArgs(action func(io.Writer) error) *cobra.Command
}

type builder struct {
	out io.Writer
	cmd cobra.Command
}

// NewCmd creates a new command builder.
func NewCmd(out io.Writer, use string) Builder {
	return &builder{
		out: out,
		cmd: cobra.Command{
			Use: use,
		},
	}
}

func (b *builder) WithDescription(description string) Builder {
	b.cmd.Short = description
	return b
}

func (b *builder) WithLongDescription(long string) Builder {
	b.cmd.Long = long
	return b
}

func (b *builder) WithCommonFlags() Builder {
	AddFlags(b.cmd.Flags(), b.cmd.Use)
	return b
}

func (b *builder) WithFlags(adder func(*pflag.FlagSet)) Builder {
	adder(b.cmd.Flags())
	return b
}

func (b *builder) ExactArgs(argCount int, action func(io.Writer, []string) error) *cobra.Command {
	b.cmd.Args = cobra.ExactArgs(argCount)
	b.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return action(b.out, args)
	}
	return &b.cmd
}

func (b *builder) NoArgs(action func(io.Writer) error) *cobra.Command {
	b.cmd.Args = cobra.NoArgs
	b.cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return action(b.out)
	}
	return &b.cmd
}
