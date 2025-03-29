package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

// BuildpackPullFlags consist of flags applicable to the `buildpack pull` command
type BuildpackPullFlags struct {
	// BuildpackRegistry is the name of the buildpack registry to use to search for
	BuildpackRegistry string
}

// BuildpackPull pulls a buildpack and stores it locally
func BuildpackPull(logger logging.Logger, cfg config.Config, pack PackClient) *cobra.Command {
	var flags BuildpackPullFlags

	cmd := &cobra.Command{
		Use:     "pull <uri>",
		Args:    cobra.ExactArgs(1),
		Short:   "Pull a buildpack from a registry and store it locally",
		Example: "pack buildpack pull example/my-buildpack@1.0.0",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			registry, err := config.GetRegistry(cfg, flags.BuildpackRegistry)
			if err != nil {
				return err
			}

			opts := client.PullBuildpackOptions{
				URI:          args[0],
				RegistryName: registry.Name,
			}

			if err := pack.PullBuildpack(cmd.Context(), opts); err != nil {
				return err
			}
			logger.Infof("Successfully pulled %s", style.Symbol(opts.URI))
			return nil
		}),
	}
	cmd.Flags().StringVarP(&flags.BuildpackRegistry, "buildpack-registry", "r", "", "Buildpack Registry name")
	AddHelpFlag(cmd, "pull")
	return cmd
}
