package cmd

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
)

func NewCmdIntegrationTest() *cobra.Command {
	return NewCmd("integrationtest").
		WithDescription("Run integrations tests in pod").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.VarP(&preBuiltImages, "images", "i", "A list of pre-built images to deploy")
			f.VarP(&buildOutputFile, "build-artifacts", "a", `Filepath containing build output.
E.g. build.out created by running skaffold build --quiet {{json .}} > build.out`)
		}).
		NoArgs(cancelWithCtrlC(context.Background(), doIntegrationTest))
}

func doIntegrationTest(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, config *latest.SkaffoldConfig) error {
		deployArtifacts := build.MergeWithPreviousBuilds(buildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts())
		r.DeployAndIntegrationTest(ctx, out, deployArtifacts)
		return nil
	})
	return nil
}
