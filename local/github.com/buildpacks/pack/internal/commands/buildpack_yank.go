package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

type BuildpackYankFlags struct {
	BuildpackRegistry string
	Undo              bool
}

func BuildpackYank(logger logging.Logger, cfg config.Config, pack PackClient) *cobra.Command {
	var flags BuildpackYankFlags

	cmd := &cobra.Command{
		Use:     "yank <buildpack-id-and-version>",
		Args:    cobra.ExactArgs(1),
		Short:   "Mark a buildpack on a Buildpack registry as unusable",
		Example: "pack yank my-buildpack@0.0.1",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			buildpackIDVersion := args[0]

			registry, err := config.GetRegistry(cfg, flags.BuildpackRegistry)
			if err != nil {
				return err
			}
			id, version, err := parseIDVersion(buildpackIDVersion)
			if err != nil {
				return err
			}

			opts := client.YankBuildpackOptions{
				ID:      id,
				Version: version,
				Type:    "github",
				URL:     registry.URL,
				Yank:    !flags.Undo,
			}

			if err := pack.YankBuildpack(opts); err != nil {
				return err
			}
			logger.Infof("Successfully yanked %s", style.Symbol(buildpackIDVersion))
			return nil
		}),
	}
	cmd.Flags().StringVarP(&flags.BuildpackRegistry, "buildpack-registry", "r", "", "Buildpack Registry name")
	cmd.Flags().BoolVarP(&flags.Undo, "undo", "u", false, "undo previously yanked buildpack")
	AddHelpFlag(cmd, "yank")

	return cmd
}

func parseIDVersion(buildpackIDVersion string) (string, string, error) {
	parts := strings.Split(buildpackIDVersion, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid buildpack id@version %s", style.Symbol(buildpackIDVersion))
	}

	return parts[0], parts[1], nil
}
