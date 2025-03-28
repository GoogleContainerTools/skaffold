package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

func NewExtensionCommand(logger logging.Logger, cfg config.Config, client PackClient, packageConfigReader PackageConfigReader) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "extension",
		Aliases: []string{"extensions"},
		Short:   "Interact with extensions",
		RunE:    nil,
	}

	cmd.AddCommand(ExtensionInspect(logger, cfg, client))
	// client and packageConfigReader to be passed later on
	cmd.AddCommand(ExtensionPackage(logger, cfg, client, packageConfigReader))
	// client to be passed later on
	cmd.AddCommand(ExtensionNew(logger))
	cmd.AddCommand(ExtensionPull(logger, cfg, client))
	cmd.AddCommand(ExtensionRegister(logger, cfg, client))
	cmd.AddCommand(ExtensionYank(logger, cfg, client))

	AddHelpFlag(cmd, "extension")
	return cmd
}
