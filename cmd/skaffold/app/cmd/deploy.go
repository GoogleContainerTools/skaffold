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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	buildOutputFile flags.BuildOutputFileFlag
	preBuiltImages  flags.Images
)

// NewCmdDeploy describes the CLI command to deploy artifacts.
func NewCmdDeploy(out io.Writer) *cobra.Command {
	return NewCmd(out, "deploy").
		WithDescription("Deploys the artifacts").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.VarP(&preBuiltImages, "images", "i", "A list of pre-built images to deploy")
			f.VarP(&buildOutputFile, "build-artifacts", "a", `Filepath containing build output.
E.g. build.out created by running skaffold build --quiet {{json .}} > build.out`)
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doDeploy))
}

func doDeploy(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, _ *latest.SkaffoldConfig) error {
		// If the BuildArtifacts contains an image in the preBuilt list,
		// use image from BuildArtifacts instead
		deployArtifacts := build.MergeWithPreviousBuilds(buildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts())

		return r.DeployAndLog(ctx, out, deployArtifacts)
	})
}
