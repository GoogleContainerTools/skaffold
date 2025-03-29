package commands

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: use BuildpackPackage instead
// PackageBuildpack packages (a) buildpack(s) into OCI format, based on a package config
func PackageBuildpack(logger logging.Logger, cfg config.Config, packager BuildpackPackager, packageConfigReader PackageConfigReader) *cobra.Command {
	var flags BuildpackPackageFlags

	cmd := &cobra.Command{
		Use:     `package-buildpack <name> --config <package-config-path>`,
		Hidden:  true,
		Args:    cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short:   "Package buildpack in OCI format.",
		Example: "pack package-buildpack my-buildpack --config ./package.toml",
		Long: "package-buildpack allows users to package (a) buildpack(s) into OCI format, which can then to be hosted in " +
			"image repositories. You can also package a number of buildpacks together, to enable easier distribution of " +
			"a set of buildpacks. Packaged buildpacks can be used as inputs to `pack build` (using the `--buildpack` flag), " +
			"and they can be included in the configs used in `pack builder create` and `pack buildpack package`. For more " +
			"on how to package a buildpack, see: https://buildpacks.io/docs/buildpack-author-guide/package-a-buildpack/.",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			deprecationWarning(logger, "package-buildpack", "buildpack package")

			if err := validateBuildpackPackageFlags(cfg, &flags); err != nil {
				return err
			}

			stringPolicy := flags.Policy
			if stringPolicy == "" {
				stringPolicy = cfg.PullPolicy
			}
			pullPolicy, err := image.ParsePullPolicy(stringPolicy)
			if err != nil {
				return errors.Wrap(err, "parsing pull policy")
			}

			cfg := pubbldpkg.DefaultConfig()
			relativeBaseDir := ""
			if flags.PackageTomlPath != "" {
				cfg, err = packageConfigReader.Read(flags.PackageTomlPath)
				if err != nil {
					return errors.Wrap(err, "reading config")
				}

				relativeBaseDir, err = filepath.Abs(filepath.Dir(flags.PackageTomlPath))
				if err != nil {
					return errors.Wrap(err, "getting absolute path for config")
				}
			}

			name := args[0]
			if err := packager.PackageBuildpack(cmd.Context(), client.PackageBuildpackOptions{
				RelativeBaseDir: relativeBaseDir,
				Name:            name,
				Format:          flags.Format,
				Config:          cfg,
				Publish:         flags.Publish,
				PullPolicy:      pullPolicy,
				Registry:        flags.BuildpackRegistry,
			}); err != nil {
				return err
			}

			action := "created"
			if flags.Publish {
				action = "published"
			}

			logger.Infof("Successfully %s package %s", action, style.Symbol(name))
			return nil
		}),
	}
	cmd.Flags().StringVarP(&flags.PackageTomlPath, "config", "c", "", "Path to package TOML config (required)")

	cmd.Flags().StringVarP(&flags.Format, "format", "f", "", `Format to save package as ("image" or "file")`)
	cmd.Flags().BoolVar(&flags.Publish, "publish", false, `Publish the buildpack directly to the container registry specified in <name>, instead of the daemon (applies to "--format=image" only).`)
	cmd.Flags().StringVar(&flags.Policy, "pull-policy", "", "Pull policy to use. Accepted values are always, never, and if-not-present. The default is always")
	cmd.Flags().StringVarP(&flags.BuildpackRegistry, "buildpack-registry", "r", "", "Buildpack Registry name")

	AddHelpFlag(cmd, "package-buildpack")
	return cmd
}
