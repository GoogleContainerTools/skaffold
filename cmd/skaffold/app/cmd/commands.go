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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
)

// Builder is used to build cobra commands.
type Builder interface {
	WithDescription(description string) Builder
	WithLongDescription(long string) Builder
	WithExample(comment, command string) Builder
	WithFlags(adder func(*pflag.FlagSet)) Builder
	WithCommonFlags() Builder
	Hidden() Builder
	ExactArgs(argCount int, action func(context.Context, io.Writer, []string) error) *cobra.Command
	NoArgs(action func(context.Context, io.Writer) error) *cobra.Command
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

func (b *builder) WithFlags(adder func(*pflag.FlagSet)) Builder {
	adder(b.cmd.Flags())
	return b
}

func (b *builder) Hidden() Builder {
	b.cmd.Hidden = true
	return b
}

func (b *builder) ExactArgs(argCount int, action func(context.Context, io.Writer, []string) error) *cobra.Command {
	b.cmd.Args = cobra.ExactArgs(argCount)
	b.cmd.RunE = func(_ *cobra.Command, args []string) error {
		err := handleWellKnownErrors(action(b.cmd.Context(), b.cmd.OutOrStdout(), args))
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
		err := handleWellKnownErrors(action(b.cmd.Context(), b.cmd.OutOrStdout()))
		// clean up server at end of the execution since post run hooks are only executed if
		// RunE is successful
		if shutdownAPIServer != nil {
			shutdownAPIServer()
		}
		return err
	}
	return &b.cmd
}

func handleWellKnownErrors(err error) error {
	if err == nil {
		return err
	}

	if aErr := sErrors.ShowAIError(err); aErr != sErrors.ErrNoSuggestionFound {
		return aErr
	}

	return err
}
