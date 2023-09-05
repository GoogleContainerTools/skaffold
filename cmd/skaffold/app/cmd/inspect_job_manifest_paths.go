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
	jobManifestPaths "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/jobManifestPaths"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

func cmdJobManifestPaths() *cobra.Command {
	return NewCmd("jobManifestPaths").
		WithDescription("View skaffold jobManifestPath information defined in the specified skaffold configuration").
		WithCommands(cmdJobManifestPathsList(), cmdJobManifestPathsModify())
}

func cmdJobManifestPathsList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of jobManifestPaths", "inspect jobManifestPaths list --format json").
		WithExample("Get list of jobManifestPaths targeting a specific configuration", "inspect jobManifestPaths list --profile local --format json").
		WithDescription("Print the list of jobManifestPaths that would be run for a given configuration (default skaffold configuration, specific module, specific profile, etc).").
		WithFlagAdder(cmdJobManifestPathsListFlags).
		NoArgs(listJobManifestPaths)
}

func cmdJobManifestPathsModify() *cobra.Command {
	return NewCmd("modify").
		WithExample("Modify the skaffold verify jobManifestPaths", "inspect jobManifestPaths list --format json").
		WithExample("Modify the jobManifestPaths targeting a specific configuration", "inspect jobManifestPaths modify --profile local --format json").
		WithDescription("Print the list of jobManifestPaths that would be run for a given configuration (default skaffold configuration, specific module, specific profile, etc).").
		WithCommonFlags().
		WithFlags([]*Flag{
			// TODO(aaron-prindle) vvv 2 commands use this, should add to common flags w/ those 2 commands added
			{Value: &outputFile, Name: "output", DefValue: "", Usage: "File to write `inspect jobManifestPath modify` result"},
		}).
		WithArgs(func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				olog.Entry(context.TODO()).Errorf("`jobManifestPaths modify` requires exactly one manifest file path argument")
				return errors.New("`jobManifestPaths modify` requires exactly one manifest file path argument")
			}
			return nil
		}, modifyJobManifestPaths)
}

func listJobManifestPaths(ctx context.Context, out io.Writer) error {
	return jobManifestPaths.PrintJobManifestPathsList(ctx, out, inspect.Options{
		Filename:          inspectFlags.filename,
		RemoteCacheDir:    inspectFlags.remoteCacheDir,
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

func modifyJobManifestPaths(ctx context.Context, out io.Writer, args []string) error {
	return jobManifestPaths.Modify(ctx, out, opts, args[0], outputFile)
}
