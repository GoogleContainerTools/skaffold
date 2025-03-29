package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use `config trusted-builders add` instead
func TrustBuilder(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "trust-builder <builder-name>",
		Args:    cobra.ExactArgs(1),
		Short:   "Trust builder",
		Long:    "Trust builder.\n\nWhen building with this builder, all lifecycle phases will be run in a single container using the builder image.",
		Example: "pack trust-builder cnbs/sample-stack-run:bionic",
		Hidden:  true,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "trust-builder", "config trusted-builders add")
			return addTrustedBuilder(args, logger, cfg, cfgPath)
		}),
	}

	AddHelpFlag(cmd, "trust-builder")
	return cmd
}
