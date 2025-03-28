package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

// ExtensionPullFlags consist of flags applicable to the `extension pull` command
type ExtensionPullFlags struct {
	// ExtensionRegistry is the name of the extension registry to use to search for
	ExtensionRegistry string
}

// ExtensionPull pulls an extension and stores it locally
func ExtensionPull(logger logging.Logger, cfg config.Config, pack PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pull <uri>",
		Args:    cobra.ExactArgs(1),
		Short:   "Pull an extension from a registry and store it locally",
		Example: "pack extension pull <extension-example>",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			// logic will be added here
			return nil
		}),
	}
	// flags will be added here
	AddHelpFlag(cmd, "pull")
	return cmd
}
