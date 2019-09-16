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
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
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
func NewCmdDeploy() *cobra.Command {
	return NewCmd("deploy").
		WithDescription("Deploy pre-built artifacts").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.VarP(&preBuiltImages, "images", "i", "A list of pre-built images to deploy")
			f.VarP(&buildOutputFile, "build-artifacts", "a", `Filepath containing build output.
E.g. build.out created by running skaffold build --quiet {{json .}} > build.out`)
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doDeploy))
}

func doDeploy(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		deployed, err := getArtifactsToDeploy(out, buildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts(), config.Build.Artifacts)
		if err != nil {
			return err
		}

		return r.DeployAndLog(ctx, out, deployed)
	})
}

func getArtifactsToDeploy(out io.Writer, fromFile, fromCLI []build.Artifact, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var deployed []build.Artifact
	for _, artifact := range artifacts {
		deployed = append(deployed, build.Artifact{
			ImageName: artifact.ImageName,
		})
	}

	// Tags provided by file take precedence over those provided on the command line
	deployed = build.MergeWithPreviousBuilds(fromCLI, deployed)
	deployed = build.MergeWithPreviousBuilds(fromFile, deployed)

	// Check that every image has a non empty tag
	for _, d := range deployed {
		if d.Tag == "" {
			tips.PrintUseRunVsDeploy(out)
			return nil, fmt.Errorf("no tag provided for image [%s]", d.ImageName)
		}
	}

	return deployed, nil
}
