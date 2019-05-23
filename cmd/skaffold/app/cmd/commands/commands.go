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

package commands

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CmdBuilder interface {
	WithDescription(description string) CmdBuilder
	WithLongDescription(long string) CmdBuilder
	WithFlags(adder func(*pflag.FlagSet)) CmdBuilder
	ExactArgs(argCount int, action func(io.Writer, []string) error) *cobra.Command
	NoArgs(action func(io.Writer) error) *cobra.Command
}

type cmdBuilder struct {
	out io.Writer
	cmd cobra.Command
}

func New(out io.Writer, use string) CmdBuilder {
	return &cmdBuilder{
		out: out,
		cmd: cobra.Command{
			Use: use,
		},
	}
}

func (c *cmdBuilder) WithDescription(description string) CmdBuilder {
	c.cmd.Short = description
	return c
}

func (c *cmdBuilder) WithLongDescription(long string) CmdBuilder {
	c.cmd.Long = long
	return c
}

func (c *cmdBuilder) WithFlags(adder func(*pflag.FlagSet)) CmdBuilder {
	adder(c.cmd.Flags())
	return c
}

func (c *cmdBuilder) ExactArgs(argCount int, action func(io.Writer, []string) error) *cobra.Command {
	c.cmd.Args = cobra.ExactArgs(argCount)
	c.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return action(c.out, args)
	}
	return &c.cmd
}

func (c *cmdBuilder) NoArgs(action func(io.Writer) error) *cobra.Command {
	c.cmd.Args = cobra.NoArgs
	c.cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return action(c.out)
	}
	return &c.cmd
}
