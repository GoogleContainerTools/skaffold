/*
Copyright 2019 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
)

var (
	dryRun bool
)

// NewCmdDelete describes the CLI command to delete deployed resources.
func NewCmdDelete() *cobra.Command {
	return NewCmd("delete").
		WithDescription("Delete any resources deployed by Skaffold").
		WithCommonFlags().
		WithExample("Print the resources to be deleted", "delete --dry-run").
		WithFlags([]*Flag{
			{Value: &dryRun, Name: "dry-run", DefValue: false, Usage: "Don't delete resources, just print them.", IsEnum: true},
		}).
		NoArgs(doDelete)
}

func doDelete(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
		manifestListByConfig, err := r.Render(ctx, io.Discard, []graph.Artifact{}, false)
		if err != nil {
			return err
		}
		return r.Cleanup(ctx, out, dryRun, manifestListByConfig)
	})
}
