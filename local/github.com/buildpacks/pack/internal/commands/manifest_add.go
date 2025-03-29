package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

// ManifestAdd adds a new image to a manifest list (image index).
func ManifestAdd(logger logging.Logger, pack PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add [OPTIONS] <manifest-list> <manifest> [flags]",
		Args:    cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
		Short:   "Add an image to a manifest list.",
		Example: `pack manifest add my-image-index my-image:some-arch`,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) (err error) {
			return pack.AddManifest(cmd.Context(), client.ManifestAddOptions{
				IndexRepoName: args[0],
				RepoName:      args[1],
			})
		}),
	}

	AddHelpFlag(cmd, "add")
	return cmd
}
