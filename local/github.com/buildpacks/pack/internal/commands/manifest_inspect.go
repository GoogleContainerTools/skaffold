package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
)

// ManifestInspect shows the manifest information stored locally
func ManifestInspect(logger logging.Logger, pack PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inspect <manifest-list>",
		Args:    cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short:   "Display information about a manifest list.",
		Example: `pack manifest inspect my-image-index`,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return errors.New("'<manifest-list>' is required")
			}
			return pack.InspectManifest(args[0])
		}),
	}

	AddHelpFlag(cmd, "inspect")
	return cmd
}
