package commands

import (
	"github.com/spf13/cobra"

	cpkg "github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

type DownloadSBOMFlags struct {
	Remote         bool
	DestinationDir string
}

func DownloadSBOM(
	logger logging.Logger,
	client PackClient,
) *cobra.Command {
	var flags DownloadSBOMFlags
	cmd := &cobra.Command{
		Use:     "download <image-name>",
		Args:    cobra.ExactArgs(1),
		Short:   "Download SBoM from specified image",
		Long:    "Download layer containing structured Software Bill of Materials (SBoM) from specified image",
		Example: "pack sbom download buildpacksio/pack",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			img := args[0]
			options := cpkg.DownloadSBOMOptions{
				Daemon:         !flags.Remote,
				DestinationDir: flags.DestinationDir,
			}

			return client.DownloadSBOM(img, options)
		}),
	}
	AddHelpFlag(cmd, "download")
	cmd.Flags().BoolVar(&flags.Remote, "remote", false, "Download SBoM of image in remote registry (without pulling image)")
	cmd.Flags().StringVarP(&flags.DestinationDir, "output-dir", "o", ".", "Path to export SBoM contents.\nIt defaults export to the current working directory.")
	return cmd
}
