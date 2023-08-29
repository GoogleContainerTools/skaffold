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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	executionModes "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/executionModes"
)

func cmdExecutionModes() *cobra.Command {
	return NewCmd("executionModes").
		WithDescription("View skaffold executionMode information defined in the specified skaffold configuration").
		WithCommands(cmdExecutionModesList())
}

func cmdExecutionModesList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of executionModes", "inspect executionModes list --format json").
		WithExample("Get list of executionModes targeting a specific configuration", "inspect executionModes list --profile local --format json").
		WithDescription("Print the list of executionModes that would be run for a given configuration (default skaffold configuration, specific module, specific profile, etc).").
		NoArgs(listExecutionModes)
}

func listExecutionModes(ctx context.Context, out io.Writer) error {
	return executionModes.PrintExecutionModesList(ctx, out, inspect.Options{
		Filename:          inspectFlags.filename,
		RemoteCacheDir:    inspectFlags.remoteCacheDir,
		OutFormat:         inspectFlags.outFormat,
		Modules:           inspectFlags.modules,
		Profiles:          inspectFlags.profiles,
		PropagateProfiles: inspectFlags.propagateProfiles,
	})
}
