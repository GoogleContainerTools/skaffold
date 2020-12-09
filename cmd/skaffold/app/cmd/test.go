/*
Copyright 2020 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// NewCmdTest describes the CLI command to test artifacts.
func NewCmdTest() *cobra.Command {
	return NewCmd("test").
		WithDescription("Test the artifacts").
		WithExample("Build the artifacts and collect the tags into a file", "build --file-output=tags.json").
		WithExample("Test those tags", "test --build-artifacts=tags.json").
		WithExample("Build the artifacts and then test them", "build -q | skaffold test --build-artifacts -").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.VarP(&fromBuildOutputFile, "build-artifacts", "a", "File containing build result from a previous 'skaffold build --file-output'")
		}).
		WithHouseKeepingMessages().
		NoArgs(doTest)
}

func doTest(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		buildArtifacts, err := getBuildArtifactsAndSetTags(r, config)
		if err != nil {
			tips.PrintTest(out)
			return err
		}

		return r.Test(ctx, out, buildArtifacts)
	})
}
