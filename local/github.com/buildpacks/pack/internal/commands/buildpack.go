package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

func NewBuildpackCommand(logger logging.Logger, cfg config.Config, client PackClient, packageConfigReader PackageConfigReader) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "buildpack",
		Aliases: []string{"buildpacks"},
		Short:   "Interact with buildpacks",
		RunE:    nil,
	}

	cmd.AddCommand(BuildpackInspect(logger, cfg, client))
	cmd.AddCommand(BuildpackPackage(logger, cfg, client, packageConfigReader))
	cmd.AddCommand(BuildpackNew(logger, client))
	cmd.AddCommand(BuildpackPull(logger, cfg, client))
	cmd.AddCommand(BuildpackRegister(logger, cfg, client))
	cmd.AddCommand(BuildpackYank(logger, cfg, client))

	AddHelpFlag(cmd, "buildpack")
	return cmd
}
