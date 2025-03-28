package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
)

func NewManifestCommand(logger logging.Logger, client PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Interact with OCI image indexes",
		Long: `An image index is a higher-level manifest which points to specific image manifests and is ideal for one or more platforms; see: https://github.com/opencontainers/image-spec/ for more details

'pack manifest' commands provide tooling to create, update, or delete images indexes or push them to a remote registry.
'pack' will save a local copy of the image index at '$PACK_HOME/manifests'; the environment variable 'XDG_RUNTIME_DIR' 
can be set to override the location, allowing manifests to be edited locally before being pushed to a registry.

These commands are experimental. For more information, consult the RFC which can be found at https://github.com/buildpacks/rfcs/blob/main/text/0124-pack-manifest-list-commands.md`,
		RunE: nil,
	}

	cmd.AddCommand(ManifestCreate(logger, client))
	cmd.AddCommand(ManifestAdd(logger, client))
	cmd.AddCommand(ManifestAnnotate(logger, client))
	cmd.AddCommand(ManifestDelete(logger, client))
	cmd.AddCommand(ManifestInspect(logger, client))
	cmd.AddCommand(ManifestPush(logger, client))
	cmd.AddCommand(ManifestRemove(logger, client))

	AddHelpFlag(cmd, "manifest")
	return cmd
}
