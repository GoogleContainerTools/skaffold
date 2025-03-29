package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use `config trusted-builders list` instead
func ListTrustedBuilders(logger logging.Logger, cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-trusted-builders",
		Short:   "List Trusted Builders",
		Long:    "List Trusted Builders.\n\nShow the builders that are either trusted by default or have been explicitly trusted locally using `trusted-builder add`",
		Example: "pack list-trusted-builders",
		Hidden:  true,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "list-trusted-builders", "config trusted-builders list")
			listTrustedBuilders(args, logger, cfg)
			return nil
		}),
	}

	AddHelpFlag(cmd, "list-trusted-builders")
	return cmd
}
