package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use `pack config registries default` instead
func SetDefaultRegistry(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	var (
		registryName string
	)

	cmd := &cobra.Command{
		Use:     "set-default-registry <name>",
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
		Short:   "Set default registry",
		Example: "pack set-default-registry myregistry",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "set-default-registry", "config registries default")
			registryName = args[0]
			if !registriesContains(config.GetRegistries(cfg), registryName) {
				return errors.Errorf("no registry with the name %s exists", style.Symbol(registryName))
			}

			if cfg.DefaultRegistryName != registryName {
				cfg.DefaultRegistryName = registryName
				err := config.Write(cfg, cfgPath)
				if err != nil {
					return err
				}
			}

			logger.Infof("Successfully set %s as the default registry", style.Symbol(registryName))

			return nil
		}),
	}
	AddHelpFlag(cmd, "set-default-registry")

	return cmd
}
