/*
Copyright 2023 The Skaffold Authors

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
	configDependencies "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/configDependencies"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

func cmdConfigDependencies() *cobra.Command {
	return NewCmd("config-dependencies").
		WithDescription("Interact with skaffold config dependency definitions.").
		WithCommands(cmdConfigDependenciesAdd())
}

func cmdConfigDependenciesAdd() *cobra.Command {
	return NewCmd("add").
		WithDescription("Add config dependencies").
		WithExample("Add config dependency defined in `dependency.json`.", "inspect config-dependencies add dependency.json -f skaffold.yaml").
		WithFlagAdder(cmdConfigDependenciesAddFlags).
		WithArgs(func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				errMsg := "`config-dependencies add` requires exactly one file path argument"
				olog.Entry(context.TODO()).Error(errMsg)
				return errors.New(errMsg)
			}
			return nil
		}, addConfigDependencies)
}

func cmdConfigDependenciesAddFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}

func addConfigDependencies(ctx context.Context, out io.Writer, args []string) error {
	return configDependencies.AddConfigDependencies(ctx, out, inspect.Options{
		Filename:       inspectFlags.filename,
		OutFormat:      inspectFlags.outFormat,
		RemoteCacheDir: inspectFlags.remoteCacheDir,
		Modules:        inspectFlags.modules,
	}, args[0])
}
