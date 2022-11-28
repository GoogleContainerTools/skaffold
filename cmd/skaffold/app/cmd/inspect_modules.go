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
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	modules "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/modules"
)

var modulesFlags = struct {
	includeAll bool
}{}

func cmdModules() *cobra.Command {
	return NewCmd("modules").
		WithDescription("Interact with configuration modules").
		WithCommands(cmdModulesList())
}

func cmdModulesList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of modules", "inspect modules list --format json").
		WithExample("Get list of all configs (including unnamed modules)", "inspect modules list -a --format json").
		WithDescription("Print the list of module names that can be invoked with the --module flag in other skaffold commands.").
		WithFlagAdder(cmdModulesListFlags).
		NoArgs(listModules)
}

func listModules(ctx context.Context, out io.Writer) error {
	return modules.PrintModulesList(ctx, out, inspect.Options{
		Filename:       inspectFlags.filename,
		RepoCacheDir:   inspectFlags.repoCacheDir,
		OutFormat:      inspectFlags.outFormat,
		ModulesOptions: inspect.ModulesOptions{IncludeAll: modulesFlags.includeAll},
	})
}

func cmdModulesListFlags(f *pflag.FlagSet) {
	f.BoolVarP(&modulesFlags.includeAll, "all", "a", false, "Include unnamed modules in the result.")
}
