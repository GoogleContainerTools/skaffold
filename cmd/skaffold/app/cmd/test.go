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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// NewCmdTest describes the CLI command to test artifacts.
func NewCmdTest() *cobra.Command {
	return NewCmd("test").
		WithDescription("Run tests against your built application images").
		WithExample("Build the artifacts and collect the tags into a file", "build --file-output=tags.json").
		WithExample("Run test against images previously built by Skaffold into a 'tags.json' file", "test --build-artifacts=tags.json").
		WithCommonFlags().
		WithHouseKeepingMessages().
		NoArgs(doTest)
}

func doTest(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, configs []*latest.SkaffoldConfig) error {
		var artifacts []*latest.Artifact
		for _, c := range configs {
			artifacts = append(artifacts, c.Build.Artifacts...)
		}
		buildArtifacts, err := getBuildArtifactsAndSetTags(r, artifacts)
		if err != nil {
			tips.PrintForTest(out)
			return err
		}

		return r.Test(ctx, out, buildArtifacts)
	})
}
