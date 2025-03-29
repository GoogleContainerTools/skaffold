package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

type BuildpackInspectFlags struct {
	Depth    int
	Registry string
	Verbose  bool
}

func BuildpackInspect(logger logging.Logger, cfg config.Config, client PackClient) *cobra.Command {
	var flags BuildpackInspectFlags
	cmd := &cobra.Command{
		Use:     "inspect <image-name>",
		Args:    cobra.ExactArgs(1),
		Short:   "Show information about a buildpack",
		Example: "pack buildpack inspect cnbs/sample-package:hello-universe",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			buildpackName := args[0]
			registry := flags.Registry
			if registry == "" {
				registry = cfg.DefaultRegistryName
			}

			return buildpackInspect(logger, buildpackName, registry, flags, cfg, client)
		}),
	}

	cmd.Flags().IntVarP(&flags.Depth, "depth", "d", -1, "Max depth to display for Detection Order.\nOmission of this flag or values < 0 will display the entire tree.")
	cmd.Flags().StringVarP(&flags.Registry, "registry", "r", "", "buildpack registry that may be searched")
	cmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "show more output")
	AddHelpFlag(cmd, "inspect")
	return cmd
}

func buildpackInspect(logger logging.Logger, buildpackName, registryName string, flags BuildpackInspectFlags, _ config.Config, pack PackClient) error {
	logger.Infof("Inspecting buildpack: %s\n", style.Symbol(buildpackName))

	inspectedBuildpacksOutput, err := inspectAllBuildpacks(
		pack,
		flags,
		client.InspectBuildpackOptions{
			BuildpackName: buildpackName,
			Daemon:        true,
			Registry:      registryName,
		},
		client.InspectBuildpackOptions{
			BuildpackName: buildpackName,
			Daemon:        false,
			Registry:      registryName,
		})
	if err != nil {
		return fmt.Errorf("error writing buildpack output: %q", err)
	}

	logger.Info(inspectedBuildpacksOutput)
	return nil
}
