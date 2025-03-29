package commands

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

func ConfigLifecycleImage(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	var unset bool

	cmd := &cobra.Command{
		Use:   "lifecycle-image <lifecycle-image>",
		Args:  cobra.MaximumNArgs(1),
		Short: "Configure a custom container image for the lifecycle",
		Long: "You can use this command to set a custom image to fetch the lifecycle from." +
			"This will be used for untrusted builders. If unset, defaults to: " + config.DefaultLifecycleImageRepo +
			"For more on trusted builders, and when to trust or untrust a builder, " +
			"check out our docs here: https://buildpacks.io/docs/tools/pack/concepts/trusted_builders/",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			switch {
			case unset:
				if len(args) > 0 {
					return errors.Errorf("lifecycle image and --unset cannot be specified simultaneously")
				}

				if cfg.LifecycleImage == "" {
					logger.Info("No custom lifecycle image was set.")
				} else {
					oldImage := cfg.LifecycleImage
					cfg.LifecycleImage = ""
					if err := config.Write(cfg, cfgPath); err != nil {
						return errors.Wrapf(err, "failed to write to config at %s", cfgPath)
					}
					logger.Infof("Successfully unset custom lifecycle image %s", style.Symbol(oldImage))
				}
			case len(args) == 0:
				if cfg.LifecycleImage != "" {
					logger.Infof("The current custom lifecycle image is %s", style.Symbol(cfg.LifecycleImage))
				} else {
					logger.Infof("No custom lifecycle image is set. Lifecycle images from %s repo will be used.", style.Symbol(config.DefaultLifecycleImageRepo))
				}
				return nil
			default:
				imageName := args[0]
				_, err := name.ParseReference(imageName)

				if err != nil {
					return errors.Wrapf(err, "Invalid image name %s provided", style.Symbol(imageName))
				}
				if imageName == cfg.LifecycleImage {
					logger.Infof("Custom lifecycle image is already set to %s", style.Symbol(imageName))
					return nil
				}

				cfg.LifecycleImage = imageName
				if err := config.Write(cfg, cfgPath); err != nil {
					return errors.Wrapf(err, "failed to write to config at %s", cfgPath)
				}
				logger.Infof("Image %s will now be used as the lifecycle image", style.Symbol(imageName))
			}

			return nil
		}),
	}

	cmd.Flags().BoolVarP(&unset, "unset", "u", false, "Unset custom lifecycle image, and use the lifecycle images from "+style.Symbol(config.DefaultLifecycleImageRepo))
	AddHelpFlag(cmd, "lifecycle-image")
	return cmd
}
