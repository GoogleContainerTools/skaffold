package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
)

// ManifestDelete deletes one or more manifest lists from local storage
func ManifestDelete(logger logging.Logger, pack PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove [manifest-list] [manifest-list...]",
		Args:    cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		Short:   "Remove one or more manifest lists from local storage",
		Example: `pack manifest remove my-image-index`,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			if err := pack.DeleteManifest(args); err != nil {
				return err
			}
			return nil
		}),
	}

	AddHelpFlag(cmd, "remove")
	return cmd
}
