package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

func NewSBOMCommand(logger logging.Logger, cfg config.Config, client PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sbom",
		Short: "Interact with SBoM",
		RunE:  nil,
	}

	cmd.AddCommand(DownloadSBOM(logger, client))
	AddHelpFlag(cmd, "sbom")
	return cmd
}
