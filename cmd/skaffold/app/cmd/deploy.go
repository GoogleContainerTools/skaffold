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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

var (
	preBuiltImages flags.Images
)

// NewCmdDeploy describes the CLI command to deploy artifacts.
func NewCmdDeploy() *cobra.Command {
	return NewCmd("deploy").
		WithDescription("Deploy pre-built artifacts").
		WithExample("Build the artifacts and collect the tags into a file", "build --file-output=tags.json").
		WithExample("Deploy those tags", "deploy --build-artifacts=tags.json").
		WithExample("Build the artifacts and then deploy them", "build -q | skaffold deploy --build-artifacts -").
		WithExample("Deploy without first rendering the manifests", "deploy --skip-render").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &preBuiltImages, Name: "images", Shorthand: "i", Usage: "A list of pre-built images to deploy"},
			{Value: &opts.SkipRender, Name: "skip-render", DefValue: false, Usage: "Don't render the manifests, just deploy them", IsEnum: true},
		}).
		WithHouseKeepingMessages().
		NoArgs(doDeploy)
}

func doDeploy(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, configs []*latest.SkaffoldConfig) error {
		if opts.SkipRender {
			return r.DeployAndLog(ctx, out, []build.Artifact{})
		}
		var artifacts []*latest.Artifact
		for _, cfg := range configs {
			artifacts = append(artifacts, cfg.Build.Artifacts...)
		}
		buildArtifacts, err := getBuildArtifactsAndSetTags(r, artifacts)
		if err != nil {
			tips.PrintUseRunVsDeploy(out)
			return err
		}

		return r.DeployAndLog(ctx, out, buildArtifacts)
	})
}
