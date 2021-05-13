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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	profiles "github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect/profiles"
)

func cmdProfiles() *cobra.Command {
	return NewCmd("profiles").
		WithDescription("Interact with configuration profiles").
		WithPersistentFlagAdder(cmdProfilesFlags).
		WithCommands(cmdProfilesList())
}

func cmdProfilesList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of profiles", "inspect profiles list --format json").
		WithExample("Get list of profiles targeting GoogleCloudBuild environment", "inspect profiles list --build-env googleCloudBuild --format json").
		WithDescription("Print the list of profile names that can be invoked with the --profile flag in other skaffold commands.").
		WithFlagAdder(cmdProfilesListFlags).
		NoArgs(listProfiles)
}

func listProfiles(ctx context.Context, out io.Writer) error {
	return profiles.PrintProfilesList(ctx, out, inspect.Options{Filename: inspectFlags.fileName, OutFormat: inspectFlags.outFormat, Modules: inspectFlags.modules, ProfilesOptions: inspect.ProfilesOptions{BuildEnv: inspect.BuildEnv(inspectFlags.buildEnv)}})
}

func cmdProfilesFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}

func cmdProfilesListFlags(f *pflag.FlagSet) {
	f.StringVar(&inspectFlags.buildEnv, "build-env", "", `If specified as one of "local", "googleCloudBuild" or "cluster", then filter the output profiles list to that build environment.`)
}
