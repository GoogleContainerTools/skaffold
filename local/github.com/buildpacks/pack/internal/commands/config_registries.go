package commands

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/slices"
	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/registry"
)

var (
	bpRegistryExplanation = "A Buildpack Registry is a (still experimental) place to publish, store, and discover buildpacks. "
)

var (
	setDefault   bool
	registryType string
)

func ConfigRegistries(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registries",
		Aliases: []string{"registry", "registreis"},
		Short:   "List, add and remove registries",
		Long:    bpRegistryExplanation + "\nYou can use the attached commands to list, add, and remove registries from your config",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			listRegistries(args, logger, cfg)
			return nil
		}),
	}

	listCmd := generateListCmd("registries", logger, cfg, listRegistries)
	listCmd.Example = "pack config registries list"
	listCmd.Long = bpRegistryExplanation + "List Registries saved in the pack config.\n\nShow the registries that are either added by default or have been explicitly added by using `pack config registries add`"
	cmd.AddCommand(listCmd)

	addCmd := generateAdd("registries", logger, cfg, cfgPath, addRegistry)
	addCmd.Args = cobra.ExactArgs(2)
	addCmd.Example = "pack config registries add my-registry https://github.com/buildpacks/my-registry"
	addCmd.Long = bpRegistryExplanation + "Users can add registries from the config by using registries remove, and publish/yank buildpacks from it, as well as use those buildpacks when building applications."
	addCmd.Flags().BoolVar(&setDefault, "default", false, "Set this buildpack registry as the default")
	addCmd.Flags().StringVar(&registryType, "type", "github", "Type of buildpack registry [git|github]")
	cmd.AddCommand(addCmd)

	rmCmd := generateRemove("registries", logger, cfg, cfgPath, removeRegistry)
	rmCmd.Example = "pack config registries remove myregistry"
	rmCmd.Long = bpRegistryExplanation + "Users can remove registries from the config by using `pack config registries remove <registry>`"
	cmd.AddCommand(rmCmd)

	cmd.AddCommand(ConfigRegistriesDefault(logger, cfg, cfgPath))

	AddHelpFlag(cmd, "registries")
	return cmd
}

func addRegistry(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	newRegistry := config.Registry{
		Name: args[0],
		URL:  args[1],
		Type: registryType,
	}

	return addRegistryToConfig(logger, newRegistry, setDefault, cfg, cfgPath)
}

func addRegistryToConfig(logger logging.Logger, newRegistry config.Registry, setDefault bool, cfg config.Config, cfgPath string) error {
	if newRegistry.Name == config.OfficialRegistryName {
		return errors.Errorf("%s is a reserved registry, please provide a different name",
			style.Symbol(config.OfficialRegistryName))
	}

	if _, ok := stringset.FromSlice(registry.Types)[newRegistry.Type]; !ok {
		return errors.Errorf("%s is not a valid type. Supported types are: %s.",
			style.Symbol(newRegistry.Type),
			strings.Join(slices.MapString(registry.Types, style.Symbol), ", "))
	}

	if registriesContains(config.GetRegistries(cfg), newRegistry.Name) {
		return errors.Errorf("Buildpack registry %s already exists.",
			style.Symbol(newRegistry.Name))
	}

	if setDefault {
		cfg.DefaultRegistryName = newRegistry.Name
	}
	cfg.Registries = append(cfg.Registries, newRegistry)
	if err := config.Write(cfg, cfgPath); err != nil {
		return errors.Wrapf(err, "writing config to %s", cfgPath)
	}

	logger.Infof("Successfully added %s to registries", style.Symbol(newRegistry.Name))
	return nil
}

func removeRegistry(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	registryName := args[0]

	if registryName == config.OfficialRegistryName {
		return errors.Errorf("%s is a reserved registry name, please provide a different registry",
			style.Symbol(config.OfficialRegistryName))
	}

	index := findRegistryIndex(cfg.Registries, registryName)
	if index < 0 {
		return errors.Errorf("registry %s does not exist", style.Symbol(registryName))
	}

	cfg.Registries = removeBPRegistry(index, cfg.Registries)
	if cfg.DefaultRegistryName == registryName {
		cfg.DefaultRegistryName = config.OfficialRegistryName
	}

	if err := config.Write(cfg, cfgPath); err != nil {
		return errors.Wrapf(err, "writing config to %s", cfgPath)
	}

	logger.Infof("Successfully removed %s from registries", style.Symbol(registryName))
	return nil
}

func listRegistries(args []string, logger logging.Logger, cfg config.Config) {
	for _, currRegistry := range config.GetRegistries(cfg) {
		isDefaultRegistry := (currRegistry.Name == cfg.DefaultRegistryName) ||
			(currRegistry.Name == config.OfficialRegistryName && cfg.DefaultRegistryName == "")

		logger.Info(fmtRegistry(
			currRegistry,
			isDefaultRegistry,
			logger.IsVerbose()))
	}
	logging.Tip(logger, "Run %s to add additional registries", style.Symbol("pack config registries add"))
}

// Local private funcs
func fmtRegistry(registry config.Registry, isDefaultRegistry, isVerbose bool) string {
	registryOutput := fmt.Sprintf("  %s", registry.Name)
	if isDefaultRegistry {
		registryOutput = fmt.Sprintf("* %s", registry.Name)
	}
	if isVerbose {
		registryOutput = fmt.Sprintf("%-12s %s", registryOutput, registry.URL)
	}

	return registryOutput
}

func registriesContains(registries []config.Registry, registry string) bool {
	return findRegistryIndex(registries, registry) != -1
}

func findRegistryIndex(registries []config.Registry, registryName string) int {
	for index, r := range registries {
		if r.Name == registryName {
			return index
		}
	}

	return -1
}

func removeBPRegistry(index int, registries []config.Registry) []config.Registry {
	registries[index] = registries[len(registries)-1]
	return registries[:len(registries)-1]
}
