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
	"errors"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	modules "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/modules"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

var modulesFlags = struct {
	includeAll bool
}{}

func cmdModules() *cobra.Command {
	return NewCmd("modules").
		WithDescription("Interact with configuration modules").
		WithCommands(cmdModulesList(), cmdModulesAdd())
}

func cmdModulesList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of modules", "inspect modules list --format json").
		WithExample("Get list of all configs (including unnamed modules)", "inspect modules list -a --format json").
		WithDescription("Print the list of module names that can be invoked with the --module flag in other skaffold commands.").
		WithFlagAdder(cmdModulesListFlags).
		NoArgs(listModules)
}

func cmdModulesAdd() *cobra.Command {
	return NewCmd("add").
		WithDescription("Add config dependencies").
		WithExample("Add config dependency defined in `depedency.json`.", "inspect modules add dependency.json -f skaffold.yaml").
		WithFlagAdder(cmdModulesAddFlags).
		WithArgs(func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				errMsg := "`config-dependencies add` requires exactly one file path argument"
				olog.Entry(context.TODO()).Errorf(errMsg)
				return errors.New(errMsg)
			}
			return nil
		}, addModules)
}

func listModules(ctx context.Context, out io.Writer) error {
	return modules.PrintModulesList(ctx, out, inspect.Options{
		Filename:       inspectFlags.filename,
		RemoteCacheDir: inspectFlags.remoteCacheDir,
		OutFormat:      inspectFlags.outFormat,
		ModulesOptions: inspect.ModulesOptions{IncludeAll: modulesFlags.includeAll},
	})
}

func addModules(ctx context.Context, out io.Writer, args []string) error {
	return modules.AddConfigDependencies(ctx, out, inspect.Options{
		Filename:       inspectFlags.filename,
		OutFormat:      inspectFlags.outFormat,
		RemoteCacheDir: inspectFlags.remoteCacheDir,
		Modules:        inspectFlags.modules,
	}, args[0])
}

func cmdModulesListFlags(f *pflag.FlagSet) {
	f.BoolVarP(&modulesFlags.includeAll, "all", "a", false, "Include unnamed modules in the result.")
}

func cmdModulesAddFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}
