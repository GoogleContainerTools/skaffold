package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

func ExtensionInspect(logger logging.Logger, cfg config.Config, client PackClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inspect <extension-name>",
		Args:    cobra.ExactArgs(1),
		Short:   "Show information about an extension",
		Example: "pack extension inspect <example-extension>",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			extensionName := args[0]
			return extensionInspect(logger, extensionName, cfg, client)
		}),
	}
	AddHelpFlag(cmd, "inspect")
	return cmd
}

func extensionInspect(logger logging.Logger, extensionName string, _ config.Config, pack PackClient) error {
	logger.Infof("Inspecting extension: %s\n", style.Symbol(extensionName))

	inspectedExtensionsOutput, err := inspectAllExtensions(
		pack,
		client.InspectExtensionOptions{
			ExtensionName: extensionName,
			Daemon:        true,
		},
		client.InspectExtensionOptions{
			ExtensionName: extensionName,
			Daemon:        false,
		})
	if err != nil {
		return fmt.Errorf("error writing extension output: %q", err)
	}

	logger.Info(inspectedExtensionsOutput)
	return nil
}
