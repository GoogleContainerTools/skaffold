package commands

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

var registryMirror string

func ConfigRegistryMirrors(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry-mirrors",
		Short:   "List, add and remove OCI registry mirrors",
		Aliases: []string{"registry-mirror"},
		Args:    cobra.MaximumNArgs(3),
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			listRegistryMirrors(args, logger, cfg)
			return nil
		}),
	}

	listCmd := generateListCmd(cmd.Use, logger, cfg, listRegistryMirrors)
	listCmd.Long = "List all registry mirrors."
	listCmd.Use = "list"
	listCmd.Example = "pack config registry-mirrors list"
	cmd.AddCommand(listCmd)

	addCmd := generateAdd("mirror for a registry", logger, cfg, cfgPath, addRegistryMirror)
	addCmd.Use = "add <registry> [-m <mirror...]"
	addCmd.Long = "Set mirror for a given registry."
	addCmd.Example = "pack config registry-mirrors add index.docker.io --mirror 10.0.0.1\npack config registry-mirrors add '*' --mirror 10.0.0.1"
	addCmd.Flags().StringVarP(&registryMirror, "mirror", "m", "", "Registry mirror")
	cmd.AddCommand(addCmd)

	rmCmd := generateRemove("mirror for a registry", logger, cfg, cfgPath, removeRegistryMirror)
	rmCmd.Use = "remove <registry>"
	rmCmd.Long = "Remove mirror for a given registry."
	rmCmd.Example = "pack config registry-mirrors remove index.docker.io"
	cmd.AddCommand(rmCmd)

	AddHelpFlag(cmd, "run-image-mirrors")
	return cmd
}

func addRegistryMirror(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	registry := args[0]
	if registryMirror == "" {
		logger.Infof("A registry mirror was not provided.")
		return nil
	}

	if cfg.RegistryMirrors == nil {
		cfg.RegistryMirrors = map[string]string{}
	}

	cfg.RegistryMirrors[registry] = registryMirror
	if err := config.Write(cfg, cfgPath); err != nil {
		return errors.Wrapf(err, "failed to write to %s", cfgPath)
	}

	logger.Infof("Registry %s configured with mirror %s", style.Symbol(registry), style.Symbol(registryMirror))
	return nil
}

func removeRegistryMirror(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	registry := args[0]
	_, ok := cfg.RegistryMirrors[registry]
	if !ok {
		logger.Infof("No registry mirror has been set for %s", style.Symbol(registry))
		return nil
	}

	delete(cfg.RegistryMirrors, registry)
	if err := config.Write(cfg, cfgPath); err != nil {
		return errors.Wrapf(err, "failed to write to %s", cfgPath)
	}

	logger.Infof("Removed mirror for %s", style.Symbol(registry))
	return nil
}

func listRegistryMirrors(args []string, logger logging.Logger, cfg config.Config) {
	if len(cfg.RegistryMirrors) == 0 {
		logger.Info("No registry mirrors have been set")
		return
	}

	buf := strings.Builder{}
	buf.WriteString("Registry Mirrors:\n")
	for registry, mirror := range cfg.RegistryMirrors {
		buf.WriteString(fmt.Sprintf("  %s: %s\n", registry, style.Symbol(mirror)))
	}

	logger.Info(buf.String())
}
