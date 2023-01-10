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
	namespaces "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/namespaces"
)

func cmdNamespaces() *cobra.Command {
	return NewCmd("namespaces").
		WithDescription("View skaffold namespace information for resources it manages").
		WithCommands(cmdNamespacesList())
}

func cmdNamespacesList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of namespaces", "inspect namespaces list --format json").
		WithExample("Get list of namespaces targeting a specific configuration", "inspect namespaces list --profile local --format json").
		WithDescription("Print the list of namespaces that would be run for a given configuration (default skaffold configuration, specific module, specific profile, etc).").
		WithFlagAdder(cmdNamespacesListFlags).
		WithArgs(func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("`inspect namespaces list` requires exactly one manifest file path argument")
			}
			return nil
		}, listNamespaces)
}

// NOTE:
//   - currently kubecontext namespaces are not handled as they were not expected for the
//     initial use cases involving this command
//   - also this code currently does not account for the possibility of the -n flag passed
//     additionally to a skaffold command (eg: skaffold apply -n foo)
func listNamespaces(ctx context.Context, out io.Writer, args []string) error {
	return namespaces.PrintNamespacesList(ctx, out, args[0], inspect.Options{
		Filename:          inspectFlags.filename,
		RepoCacheDir:      inspectFlags.repoCacheDir,
		OutFormat:         inspectFlags.outFormat,
		Modules:           inspectFlags.modules,
		Profiles:          inspectFlags.profiles,
		PropagateProfiles: inspectFlags.propagateProfiles,
	})
}

func cmdNamespacesListFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.profiles, "profile", "p", nil, `Profile names to activate`)
	f.BoolVar(&inspectFlags.propagateProfiles, "propagate-profiles", true, `Setting '--propagate-profiles=false' disables propagating profiles set by the '--profile' flag across config dependencies. This mean that only profiles defined directly in the target 'skaffold.yaml' file are activated.`)
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}
