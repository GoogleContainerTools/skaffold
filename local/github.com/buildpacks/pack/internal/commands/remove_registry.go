package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use config registries remove instead
func RemoveRegistry(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove-registry <name>",
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
		Short:   "Remove registry",
		Example: "pack remove-registry myregistry",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "remove-registry", "config registries remove")
			return removeRegistry(args, logger, cfg, cfgPath)
		}),
	}

	AddHelpFlag(cmd, "remove-registry")
	return cmd
}
