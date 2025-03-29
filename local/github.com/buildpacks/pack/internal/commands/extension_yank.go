package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

type ExtensionYankFlags struct {
	ExtensionRegistry string
	Undo              bool
}

func ExtensionYank(logger logging.Logger, cfg config.Config, pack PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "yank <extension-id-and-version>",
		Args:    cobra.ExactArgs(1),
		Short:   "Yank an extension from a registry",
		Example: "pack yank <extension-example>",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			// logic will be added here
			return nil
		}),
	}
	// flags will be added here
	AddHelpFlag(cmd, "yank")

	return cmd
}
