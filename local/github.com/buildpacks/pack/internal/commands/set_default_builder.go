package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use `pack config default-builder`
func SetDefaultBuilder(logger logging.Logger, cfg config.Config, cfgPath string, client PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-default-builder <builder-name>",
		Hidden:  true,
		Args:    cobra.MaximumNArgs(1),
		Short:   "Set default builder used by other commands",
		Long:    "Set default builder used by other commands.\n\n** For suggested builders simply leave builder name empty. **",
		Example: "pack set-default-builder cnbs/sample-builder:bionic",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "set-default-builder", "config default-builder")
			if len(args) < 1 || args[0] == "" {
				logger.Infof("Usage:\n\t%s\n", cmd.UseLine())
				suggestBuilders(logger, client)
				return nil
			}

			imageName := args[0]

			logger.Debug("Verifying local image...")
			info, err := client.InspectBuilder(imageName, true)
			if err != nil {
				return err
			}

			if info == nil {
				logger.Debug("Verifying remote image...")
				info, err := client.InspectBuilder(imageName, false)
				if err != nil {
					return err
				}

				if info == nil {
					return fmt.Errorf("builder %s not found", style.Symbol(imageName))
				}
			}

			cfg.DefaultBuilder = imageName
			if err := config.Write(cfg, cfgPath); err != nil {
				return err
			}
			logger.Infof("Builder %s is now the default builder", style.Symbol(imageName))
			return nil
		}),
	}

	AddHelpFlag(cmd, "set-default-builder")
	return cmd
}
