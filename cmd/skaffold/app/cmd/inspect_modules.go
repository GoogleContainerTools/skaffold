/*
Copyright 2021 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
)

func cmdModules() *cobra.Command {
	return NewCmd("modules").
		WithDescription("Interact with configuration modules").
		WithCommands(cmdModulesList())
}

func cmdModulesList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of modules", "skaffold inspect modules list --format json").
		WithDescription("Print the list of module names that can be invoked with the --module flag in other skaffold commands.").
		NoArgs(listModules)
}

func listModules(ctx context.Context, out io.Writer) error {
	return inspect.PrintModulesList(ctx, out, inspect.Options{Filename: inspectFlags.fileName, OutFormat: inspectFlags.outFormat})
}
