package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

type ExtensionRegisterFlags struct {
	ExtensionRegistry string
}

func ExtensionRegister(logger logging.Logger, cfg config.Config, pack PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "register <image>",
		Args:    cobra.ExactArgs(1),
		Short:   "Register an extension to a registry",
		Example: "pack extension register <extension-example>",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			// logic will be added here
			return nil
		}),
	}
	// flags will be added here
	AddHelpFlag(cmd, "register")
	return cmd
}
