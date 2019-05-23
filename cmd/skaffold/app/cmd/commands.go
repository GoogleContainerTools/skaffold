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

type Builder interface {
	WithDescription(description string) Builder
	WithLongDescription(long string) Builder
	WithCommonFlags() Builder
	WithFlags(adder func(*pflag.FlagSet)) Builder
	ExactArgs(argCount int, action func(io.Writer, []string) error) *cobra.Command
	NoArgs(action func(io.Writer) error) *cobra.Command
}

type builder struct {
	out io.Writer
	cmd cobra.Command
}

func NewCmd(out io.Writer, use string) Builder {
	return &builder{
		out: out,
		cmd: cobra.Command{
			Use: use,
		},
	}
}

func (c *builder) WithDescription(description string) Builder {
	c.cmd.Short = description
	return c
}

func (c *builder) WithLongDescription(long string) Builder {
	c.cmd.Long = long
	return c
}

func (c *builder) WithCommonFlags() Builder {
	AddFlags(c.cmd.Flags(), c.cmd.Use)
	return c
}

func (c *builder) WithFlags(adder func(*pflag.FlagSet)) Builder {
	adder(c.cmd.Flags())
	return c
}

func (c *builder) ExactArgs(argCount int, action func(io.Writer, []string) error) *cobra.Command {
	c.cmd.Args = cobra.ExactArgs(argCount)
	c.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return action(c.out, args)
	}
	return &c.cmd
}

func (c *builder) NoArgs(action func(io.Writer) error) *cobra.Command {
	c.cmd.Args = cobra.NoArgs
	c.cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return action(c.out)
	}
	return &c.cmd
}
