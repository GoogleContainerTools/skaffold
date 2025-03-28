package commands

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

// ManifestPushFlags define flags provided to the ManifestPush
type ManifestPushFlags struct {
	format          string
	insecure, purge bool
}

// ManifestPush pushes a manifest list to a remote registry.
func ManifestPush(logger logging.Logger, pack PackClient) *cobra.Command {
	var flags ManifestPushFlags

	cmd := &cobra.Command{
		Use:     "push [OPTIONS] <manifest-list> [flags]",
		Args:    cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short:   "Push a manifest list to a registry.",
		Example: `pack manifest push my-image-index`,
		Long: `manifest push' pushes a manifest list to a registry.
Use other 'pack manifest' commands to prepare the manifest list locally, then use the push command.`,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			format, err := parseFormatFlag(strings.ToLower(flags.format))
			if err != nil {
				return err
			}

			return pack.PushManifest(client.PushManifestOptions{
				IndexRepoName: args[0],
				Format:        format,
				Insecure:      flags.insecure,
				Purge:         flags.purge,
			})
		}),
	}

	cmd.Flags().StringVarP(&flags.format, "format", "f", "oci", "Media type to use when saving the image index. Accepted values are: oci, docker")
	cmd.Flags().BoolVar(&flags.insecure, "insecure", false, "When pushing the index to a registry, do not use TLS encryption or certificate verification")
	cmd.Flags().BoolVar(&flags.purge, "purge", false, "Delete the manifest list from local storage if pushing succeeds")

	AddHelpFlag(cmd, "push")
	return cmd
}
