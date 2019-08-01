package cmd

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/spf13/cobra"
	"io"
)

func NewCmdIntegrationTest() *cobra.Command {
	return NewCmd("integrationtest").
		WithDescription("Run integrations tests in pod").
		WithCommonFlags().
		NoArgs(cancelWithCtrlC(context.Background(), doIntegrationTest))
}

func doIntegrationTest(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, _ *latest.SkaffoldConfig) error {
		return r.ExecIntegrationTest(ctx, out)
	})
}
