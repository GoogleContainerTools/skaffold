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
	tests "github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect/tests"
)

func cmdTests() *cobra.Command {
	return NewCmd("tests").
		WithDescription("View skaffold test information").
		WithPersistentFlagAdder(cmdTestsFlags).
		WithCommands(cmdTestsList())
}

func cmdTestsList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of tests", "inspect tests list --format json").
		WithExample("Get list of tests targeting a specific configuration", "inspect tests list --profile local --format json").
		WithDescription("Print the list of tests that would be run for a given configuration (default skaffold configuration, specific module, specific profile, etc).").
		NoArgs(listTests)
}

func listTests(ctx context.Context, out io.Writer) error {
	return tests.PrintTestsList(ctx, out, inspect.Options{
		Filename:     inspectFlags.filename,
		RepoCacheDir: inspectFlags.repoCacheDir,
		OutFormat:    inspectFlags.outFormat,
		Modules:      inspectFlags.modules,
	})
}

func cmdTestsFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
	f.StringSliceVarP(&inspectFlags.profiles, "profile", "p", nil, `Profile names to activate`)
}
