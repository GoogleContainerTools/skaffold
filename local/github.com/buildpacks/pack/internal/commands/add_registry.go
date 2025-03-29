package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use config registries add instead
func AddBuildpackRegistry(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	var (
		setDefault   bool
		registryType string
	)

	cmd := &cobra.Command{
		Use:     "add-registry <name> <url>",
		Args:    cobra.ExactArgs(2),
		Hidden:  true,
		Short:   "Add buildpack registry to your pack config file",
		Example: "pack add-registry my-registry https://github.com/buildpacks/my-registry",
		Long: "A Buildpack Registry is a (still experimental) place to publish, store, and discover buildpacks. " +
			"Users can add buildpacks registries using add-registry, and publish/yank buildpacks from it, as well as use those buildpacks when building applications.",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "add-registry", "config registries add")
			newRegistry := config.Registry{
				Name: args[0],
				URL:  args[1],
				Type: registryType,
			}

			return addRegistryToConfig(logger, newRegistry, setDefault, cfg, cfgPath)
		}),
	}
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set this buildpack registry as the default")
	cmd.Flags().StringVar(&registryType, "type", "github", "Type of buildpack registry [git|github]")
	AddHelpFlag(cmd, "add-registry")

	return cmd
}
