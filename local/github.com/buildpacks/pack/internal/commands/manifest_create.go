package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

// ManifestCreateFlags define flags provided to the ManifestCreate
type ManifestCreateFlags struct {
	format            string
	insecure, publish bool
}

// ManifestCreate creates an image index for a multi-arch image
func ManifestCreate(logger logging.Logger, pack PackClient) *cobra.Command {
	var flags ManifestCreateFlags

	cmd := &cobra.Command{
		Use:     "create <manifest-list> <manifest> [<manifest> ... ] [flags]",
		Args:    cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
		Short:   "Create a new manifest list.",
		Example: `pack manifest create my-image-index my-image:some-arch my-image:some-other-arch`,
		Long:    `Create a new manifest list (e.g., for multi-arch images) which will be stored locally for manipulating images within the index`,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			format, err := parseFormatFlag(strings.ToLower(flags.format))
			if err != nil {
				return err
			}

			if err = validateCreateManifestFlags(flags); err != nil {
				return err
			}

			return pack.CreateManifest(
				cmd.Context(),
				client.CreateManifestOptions{
					IndexRepoName: args[0],
					RepoNames:     args[1:],
					Format:        format,
					Insecure:      flags.insecure,
					Publish:       flags.publish,
				},
			)
		}),
	}

	cmdFlags := cmd.Flags()

	cmdFlags.StringVarP(&flags.format, "format", "f", "oci", "Media type to use when saving the image index. Accepted values are: oci, docker")
	cmdFlags.BoolVar(&flags.insecure, "insecure", false, "When pushing the index to a registry, do not use TLS encryption or certificate verification; use with --publish")
	cmdFlags.BoolVar(&flags.publish, "publish", false, "Publish directly to a registry without saving a local copy")

	AddHelpFlag(cmd, "create")
	return cmd
}

func validateCreateManifestFlags(flags ManifestCreateFlags) error {
	if flags.insecure && !flags.publish {
		return fmt.Errorf("insecure flag requires the publish flag")
	}
	return nil
}
