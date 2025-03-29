package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use `pack config run-image-mirrors add` instead
// SetRunImagesMirrors sets run image mirros for a given run image
func SetRunImagesMirrors(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	var mirrors []string

	cmd := &cobra.Command{
		Use:     "set-run-image-mirrors <run-image-name> --mirror <run-image-mirror>",
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
		Short:   "Set mirrors to other repositories for a given run image",
		Example: "pack set-run-image-mirrors cnbs/sample-stack-run:bionic --mirror index.docker.io/cnbs/sample-stack-run:bionic",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "set-run-image-mirrors", "config run-image-mirrors")
			runImage := args[0]
			cfg = config.SetRunImageMirrors(cfg, runImage, mirrors)
			if err := config.Write(cfg, cfgPath); err != nil {
				return err
			}

			for _, mirror := range mirrors {
				logger.Infof("Run Image %s configured with mirror %s", style.Symbol(runImage), style.Symbol(mirror))
			}
			if len(mirrors) == 0 {
				logger.Infof("All mirrors removed for Run Image %s", style.Symbol(runImage))
			}
			return nil
		}),
	}
	cmd.Flags().StringSliceVarP(&mirrors, "mirror", "m", nil, "Run image mirror"+stringSliceHelp("mirror"))
	AddHelpFlag(cmd, "set-run-image-mirrors")
	return cmd
}
