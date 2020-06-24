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
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/schema"
)

func NewCmdSchema() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "List and print json schemas used to validate skaffold.yaml configuration",
	}

	cmd.AddCommand(NewCmdSchemaGet())
	cmd.AddCommand(NewCmdSchemaList())
	return cmd
}

func NewCmdSchemaList() *cobra.Command {
	return NewCmd("list").
		WithDescription("List skaffold.yaml's json schema versions").
		WithExample("List all the versions", "schema list").
		WithExample("List all the versions, in json format", "schema list -o json").
		WithFlags(func(f *pflag.FlagSet) {
			f.StringVarP(&schema.OutputType, "output", "o", "plain", "Type of output: `plain` or `json`.")
		}).
		NoArgs(schema.List)
}

func NewCmdSchemaGet() *cobra.Command {
	return NewCmd("get").
		WithDescription("Print a given skaffold.yaml's json schema").
		WithExample("Print the schema in version `skaffold/v1`", "schema get skaffold/v1").
		ExactArgs(1, func(_ context.Context, out io.Writer, args []string) error {
			version := args[0]
			return schema.Print(out, version)
		})
}
