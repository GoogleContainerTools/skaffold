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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/commands"
	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	buildOutputFile flags.BuildOutputFileFlag
	preBuiltImages  flags.Images
)

// NewCmdDeploy describes the CLI command to deploy artifacts.
func NewCmdDeploy(out io.Writer) *cobra.Command {
	return commands.
		New(out).
		WithDescription("deploy", "Deploys the artifacts").
		WithFlags(func(f *pflag.FlagSet) {
			f.VarP(&preBuiltImages, "images", "i", "A list of pre-built images to deploy")
			f.VarP(&buildOutputFile, "build-artifacts", "a", `Filepath containing build output.
E.g. build.out created by running skaffold build --quiet {{json .}} > build.out`)
			AddRunDevFlags(f)
			AddRunDeployFlags(f)
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doDeploy))
}

func doDeploy(ctx context.Context, out io.Writer) error {
	runner, _, err := newRunner(opts)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}
	defer runner.RPCServerShutdown()

	// If the BuildArtifacts contains an image in the preBuilt list,
	// use image from BuildArtifacts instead
	deployArtifacts := build.MergeWithPreviousBuilds(buildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts())

	return runner.Deploy(ctx, out, deployArtifacts)
}
