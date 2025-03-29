package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

func ConfigRegistriesDefault(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	var unset bool

	cmd := &cobra.Command{
		Use:     "default <name>",
		Args:    cobra.MaximumNArgs(1),
		Short:   "Set default registry",
		Example: "pack config registries default myregistry",
		Long: bpRegistryExplanation + "\n\nYou can use this command to list, set, and unset a default registry, which will be used when looking for buildpacks:" +
			"* To list your default registry, run `pack config registries default`.\n" +
			"* To set your default registry, run `pack config registries default <registry-name>`.\n" +
			"* To unset your default registry, run `pack config registries default --unset`.\n" +
			fmt.Sprintf("Unsetting the default registry will reset the default-registry to be the official buildpacks registry, %s", style.Symbol(config.DefaultRegistry().URL)),
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			switch {
			case unset:
				if cfg.DefaultRegistryName == "" || cfg.DefaultRegistryName == config.OfficialRegistryName {
					return errors.Errorf("Registry %s is a protected registry, and can be replaced as a default registry, but not removed entirely. "+
						"To add a new registry and set as default, run `pack config registries add <registry-name> <registry-url> --default.\n"+
						"To set an existing registry as default, call `pack config registries default <registry-name>`", style.Symbol(config.OfficialRegistryName))
				}
				oldRegistry := cfg.DefaultRegistryName
				cfg.DefaultRegistryName = ""
				if err := config.Write(cfg, cfgPath); err != nil {
					return errors.Wrapf(err, "writing config to %s", cfgPath)
				}
				logger.Infof("Successfully unset default registry %s", style.Symbol(oldRegistry))
				logger.Infof("Default registry has been set to %s", style.Symbol(config.OfficialRegistryName))
			case len(args) == 0: // list
				if cfg.DefaultRegistryName == "" {
					cfg.DefaultRegistryName = config.OfficialRegistryName
				}
				logger.Infof("The current default registry is %s", style.Symbol(cfg.DefaultRegistryName))
			default: // set
				registryName := args[0]
				if !registriesContains(config.GetRegistries(cfg), registryName) {
					return errors.Errorf("no registry with the name %s exists", style.Symbol(registryName))
				}

				if cfg.DefaultRegistryName != registryName {
					cfg.DefaultRegistryName = registryName
					err := config.Write(cfg, cfgPath)
					if err != nil {
						return errors.Wrapf(err, "writing config to %s", cfgPath)
					}
				}

				logger.Infof("Successfully set %s as the default registry", style.Symbol(registryName))
			}

			return nil
		}),
	}

	cmd.Flags().BoolVarP(&unset, "unset", "u", false, "Unset the current default registry, and set it to the official buildpacks registry")
	AddHelpFlag(cmd, "default")
	return cmd
}
