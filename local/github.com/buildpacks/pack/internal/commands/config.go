package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

func NewConfigCommand(logger logging.Logger, cfg config.Config, cfgPath string, client PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Interact with your local pack config file",
		RunE:  nil,
	}

	cmd.AddCommand(ConfigDefaultBuilder(logger, cfg, cfgPath, client))
	cmd.AddCommand(ConfigExperimental(logger, cfg, cfgPath))
	cmd.AddCommand(ConfigPullPolicy(logger, cfg, cfgPath))
	cmd.AddCommand(ConfigRegistries(logger, cfg, cfgPath))
	cmd.AddCommand(ConfigRunImagesMirrors(logger, cfg, cfgPath))
	cmd.AddCommand(ConfigTrustedBuilder(logger, cfg, cfgPath))
	cmd.AddCommand(ConfigLifecycleImage(logger, cfg, cfgPath))
	cmd.AddCommand(ConfigRegistryMirrors(logger, cfg, cfgPath))

	AddHelpFlag(cmd, "config")
	return cmd
}

type editCfgFunc func(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error

func generateAdd(cmdName string, logger logging.Logger, cfg config.Config, cfgPath string, addFunc editCfgFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Args:  cobra.ExactArgs(1),
		Short: fmt.Sprintf("Add a %s", cmdName),
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			return addFunc(args, logger, cfg, cfgPath)
		}),
	}

	return cmd
}

func generateRemove(cmdName string, logger logging.Logger, cfg config.Config, cfgPath string, rmFunc editCfgFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Args:  cobra.ExactArgs(1),
		Short: fmt.Sprintf("Remove a %s", cmdName),
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			return rmFunc(args, logger, cfg, cfgPath)
		}),
	}

	return cmd
}

type listFunc func(args []string, logger logging.Logger, cfg config.Config)

func generateListCmd(cmdName string, logger logging.Logger, cfg config.Config, listFunc listFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Args:  cobra.MaximumNArgs(1),
		Short: fmt.Sprintf("List %s", cmdName),
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			listFunc(args, logger, cfg)
			return nil
		}),
	}

	return cmd
}
