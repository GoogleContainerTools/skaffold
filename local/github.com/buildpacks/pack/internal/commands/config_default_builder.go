package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

var suggestedBuilderString = "For suggested builders, run `pack builder suggest`."

func ConfigDefaultBuilder(logger logging.Logger, cfg config.Config, cfgPath string, client PackClient) *cobra.Command {
	var unset bool

	cmd := &cobra.Command{
		Use:   "default-builder",
		Args:  cobra.MaximumNArgs(1),
		Short: "List, set and unset the default builder used by other commands",
		Long: "List, set, and unset the default builder used by other commands.\n\n" +
			"* To list your default builder, run `pack config default-builder`.\n" +
			"* To set your default builder, run `pack config default-builder <builder-name>`.\n" +
			"* To unset your default builder, run `pack config default-builder --unset`.\n\n" +
			suggestedBuilderString,
		Example: "pack config default-builder cnbs/sample-builder:bionic",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			switch {
			case unset:
				if cfg.DefaultBuilder == "" {
					logger.Info("No default builder was set")
				} else {
					oldBuilder := cfg.DefaultBuilder
					cfg.DefaultBuilder = ""
					if err := config.Write(cfg, cfgPath); err != nil {
						return errors.Wrapf(err, "failed to write to config at %s", cfgPath)
					}
					logger.Infof("Successfully unset default builder %s", style.Symbol(oldBuilder))
				}
			case len(args) == 0:
				if cfg.DefaultBuilder != "" {
					logger.Infof("The current default builder is %s", style.Symbol(cfg.DefaultBuilder))
				} else {
					logger.Infof("No default builder is set. \n\n%s", suggestedBuilderString)
				}
				return nil
			default:
				imageName := args[0]
				if err := validateBuilderExists(logger, imageName, client); err != nil {
					return errors.Wrapf(err, "validating that builder %s exists", style.Symbol(imageName))
				}

				cfg.DefaultBuilder = imageName
				if err := config.Write(cfg, cfgPath); err != nil {
					return errors.Wrapf(err, "failed to write to config at %s", cfgPath)
				}
				logger.Infof("Builder %s is now the default builder", style.Symbol(imageName))
			}

			return nil
		}),
	}

	cmd.Flags().BoolVarP(&unset, "unset", "u", false, "Unset the current default builder")
	AddHelpFlag(cmd, "config default-builder")
	return cmd
}

func validateBuilderExists(logger logging.Logger, imageName string, client PackClient) error {
	logger.Debug("Verifying local image...")
	info, err := client.InspectBuilder(imageName, true)
	if err != nil {
		return err
	}

	if info == nil {
		logger.Debug("Verifying remote image...")
		info, err := client.InspectBuilder(imageName, false)
		if err != nil {
			return errors.Wrapf(err, "failed to inspect remote image %s", style.Symbol(imageName))
		}

		if info == nil {
			return fmt.Errorf("builder %s not found", style.Symbol(imageName))
		}
	}

	return nil
}
