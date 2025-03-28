package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
)

// ManifestRemove will remove the specified image manifest if it is already referenced in the index
func ManifestRemove(logger logging.Logger, pack PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm [manifest-list] [manifest] [manifest...] [flags]",
		Args:    cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
		Short:   "Remove an image manifest from a manifest list.",
		Example: `pack manifest rm my-image-index my-image@sha256:<some-sha>`,
		Long: `'manifest rm' will remove the specified image manifest if it is already referenced in the index.
Users must pass the digest of the image in order to delete it from the index.
To discard __all__ the images in an index and the index itself, use 'manifest delete'.`,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			if err := pack.RemoveManifest(args[0], args[1:]); err != nil {
				return err
			}
			return nil
		}),
	}

	AddHelpFlag(cmd, "rm")
	return cmd
}
