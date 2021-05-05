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
)

func cmdBuildEnv() *cobra.Command {
	return NewCmd("build-env").
		WithDescription("Interact with skaffold build environment definitions.").
		WithPersistentFlagAdder(cmdBuildEnvFlags).
		WithCommands(cmdBuildEnvList())
}

func cmdBuildEnvList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of target build environments with activated profiles p1 and p2", "inspect build-env list -p p1,p2 --format json").
		WithDescription("Print the list of active build environments.").
		WithFlagAdder(cmdBuildEnvListFlags).
		NoArgs(listBuildEnv)
}

func listBuildEnv(ctx context.Context, out io.Writer) error {
	return inspect.PrintBuildEnvsList(ctx, out, inspect.Options{Filename: inspectFlags.fileName, OutFormat: inspectFlags.outFormat, Modules: inspectFlags.modules, BuildEnvOptions: inspect.BuildEnvOptions{Profiles: inspectFlags.profiles}})
}

func cmdBuildEnvFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}

func cmdBuildEnvListFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.profiles, "profile", "p", nil, `Profile names to activate`)
}
