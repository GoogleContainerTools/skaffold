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
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/embed"
)

// Builder is used to build cobra commands.
type Builder interface {
	WithArgs(cobra.PositionalArgs, func(context.Context, io.Writer, []string) error) *cobra.Command
	WithDescription(description string) Builder
	WithLongDescription(long string) Builder
	WithExample(comment, command string) Builder
	WithFlagAdder(adder func(*pflag.FlagSet)) Builder
	WithFlags([]*Flag) Builder
	WithHouseKeepingMessages() Builder
	WithCommonFlags() Builder
	Hidden() Builder
	ExactArgs(argCount int, action func(context.Context, io.Writer, []string) error) *cobra.Command
	NoArgs(action func(context.Context, io.Writer) error) *cobra.Command
	WithCommands(cmds ...*cobra.Command) *cobra.Command
	WithPersistentFlagAdder(adder func(*pflag.FlagSet)) Builder
	WithKpt() Builder
}

type builder struct {
	cmd cobra.Command
}

// NewCmd creates a new command builder.
func NewCmd(use string) Builder {
	return &builder{
		cmd: cobra.Command{
			Use: use,
		},
	}
}

// WithKpt makes sure the `kpt` is executable, otherwise it installs a builtin `kpt`.
func (b *builder) WithKpt() Builder {
	_, err := exec.LookPath("kpt")
	if err != nil {
		e := embed.UseBuiltinKpt()
		if e != nil {
			b.cmd.RunE = func(_ *cobra.Command, args []string) error {
				return e
			}
		}
	}
	return b
}

func (b *builder) WithDescription(description string) Builder {
	b.cmd.Short = description
	return b
}

func (b *builder) WithLongDescription(long string) Builder {
	b.cmd.Long = long
	return b
}

func (b *builder) WithExample(comment, command string) Builder {
	if b.cmd.Example != "" {
		b.cmd.Example += "\n"
	}
	b.cmd.Example += fmt.Sprintf("  # %s\n  skaffold %s\n", comment, command)
	return b
}

func (b *builder) WithCommonFlags() Builder {
	AddFlags(&b.cmd)
	return b
}

func (b *builder) WithHouseKeepingMessages() Builder {
	allowHouseKeepingMessages(&b.cmd)
	return b
}

func (b *builder) WithFlagAdder(adder func(*pflag.FlagSet)) Builder {
	adder(b.cmd.Flags())
	return b
}

func (b *builder) WithFlags(flags []*Flag) Builder {
	for _, f := range flags {
		fl := f.flag(b.cmd.Use)
		b.cmd.Flags().AddFlag(fl)
	}
	b.cmd.PreRun = func(cmd *cobra.Command, args []string) {
		ResetFlagDefaults(cmd, flags)
	}
	return b
}

func (b *builder) WithPersistentFlagAdder(adder func(*pflag.FlagSet)) Builder {
	adder(b.cmd.PersistentFlags())
	return b
}

func (b *builder) Hidden() Builder {
	b.cmd.Hidden = true
	return b
}

func (b *builder) ExactArgs(argCount int, action func(context.Context, io.Writer, []string) error) *cobra.Command {
	b.cmd.Args = cobra.ExactArgs(argCount)
	b.cmd.RunE = func(_ *cobra.Command, args []string) error {
		err := action(b.cmd.Context(), b.cmd.OutOrStdout(), args)
		// clean up server at end of the execution since post run hooks are only executed if
		// RunE is successful
		if shutdownAPIServer != nil {
			shutdownAPIServer()
		}
		return err
	}
	return &b.cmd
}

func (b *builder) NoArgs(action func(context.Context, io.Writer) error) *cobra.Command {
	b.cmd.Args = cobra.NoArgs
	b.cmd.RunE = func(*cobra.Command, []string) error {
		err := action(b.cmd.Context(), b.cmd.OutOrStdout())
		// clean up server at end of the execution since post run hooks are only executed if
		// RunE is successful
		if shutdownAPIServer != nil {
			shutdownAPIServer()
		}
		return err
	}
	return &b.cmd
}

func (b *builder) WithArgs(f cobra.PositionalArgs, action func(context.Context, io.Writer, []string) error) *cobra.Command {
	b.cmd.Args = f
	b.cmd.RunE = func(_ *cobra.Command, args []string) error {
		err := action(b.cmd.Context(), b.cmd.OutOrStdout(), args)
		// clean up server at end of the execution since post run hooks are only executed if
		// RunE is successful
		if shutdownAPIServer != nil {
			shutdownAPIServer()
		}
		return err
	}
	return &b.cmd
}

func (b *builder) WithCommands(cmds ...*cobra.Command) *cobra.Command {
	for _, c := range cmds {
		b.cmd.AddCommand(c)
	}
	return &b.cmd
}
