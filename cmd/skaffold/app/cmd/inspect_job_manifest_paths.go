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
	jobManifestPaths "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/jobManifestPaths"
)

func cmdJobManifestPaths() *cobra.Command {
	return NewCmd("jobManifestPaths").
		WithDescription("View skaffold jobManifestPath information for resources it manages").
		WithCommands(cmdJobManifestPathsList())
}

func cmdJobManifestPathsList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of jobManifestPaths", "inspect jobManifestPaths list --format json").
		WithExample("Get list of jobManifestPaths targeting a specific configuration", "inspect jobManifestPaths list --profile local --format json").
		WithDescription("Print the list of jobManifestPaths that would be run for a given configuration (default skaffold configuration, specific module, specific profile, etc).").
		WithFlagAdder(cmdJobManifestPathsListFlags).
		NoArgs(listJobManifestPaths)
}

func listJobManifestPaths(ctx context.Context, out io.Writer) error {
	return jobManifestPaths.PrintJobManifestPathsList(ctx, out, inspect.Options{
		Filename:          inspectFlags.filename,
		RepoCacheDir:      inspectFlags.repoCacheDir,
		OutFormat:         inspectFlags.outFormat,
		Modules:           inspectFlags.modules,
		Profiles:          inspectFlags.profiles,
		PropagateProfiles: inspectFlags.propagateProfiles,
	})
}

func cmdJobManifestPathsListFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.profiles, "profile", "p", nil, `Profile names to activate`)
	f.BoolVar(&inspectFlags.propagateProfiles, "propagate-profiles", true, `Setting '--propagate-profiles=false' disables propagating profiles set by the '--profile' flag across config dependencies. This mean that only profiles defined directly in the target 'skaffold.yaml' file are activated.`)
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}
