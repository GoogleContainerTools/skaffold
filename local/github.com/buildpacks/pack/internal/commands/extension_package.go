package commands

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
)

// ExtensionPackageFlags define flags provided to the ExtensionPackage command
type ExtensionPackageFlags struct {
	PackageTomlPath string
	Format          string
	Targets         []string
	Publish         bool
	Policy          string
	Path            string
}

// ExtensionPackager packages extensions
type ExtensionPackager interface {
	PackageExtension(ctx context.Context, options client.PackageBuildpackOptions) error
}

// ExtensionPackage packages (a) extension(s) into OCI format, based on a package config
func ExtensionPackage(logger logging.Logger, cfg config.Config, packager ExtensionPackager, packageConfigReader PackageConfigReader) *cobra.Command {
	var flags ExtensionPackageFlags
	cmd := &cobra.Command{
		Use:     "package <name> --config <config-path>",
		Short:   "Package an extension in OCI format",
		Args:    cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Example: "pack extension package /output/file.cnb --path /extracted/from/tgz/folder --format file\npack extension package registry/image-name --path  /extracted/from/tgz/folder --format image --publish",
		Long: "extension package allows users to package (an) extension(s) into OCI format, which can then to be hosted in " +
			"image repositories or persisted on disk as a '.cnb' file." +
			"Packaged extensions can be used as inputs to `pack build` (using the `--extension` flag), " +
			"and they can be included in the configs used in `pack builder create` and `pack extension package`. For more " +
			"on how to package an extension, see: https://buildpacks.io/docs/buildpack-author-guide/package-a-buildpack/.",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			if err := validateExtensionPackageFlags(&flags); err != nil {
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

			exPackageCfg := pubbldpkg.DefaultExtensionConfig()
			var exPath string
			if flags.Path != "" {
				if exPath, err = filepath.Abs(flags.Path); err != nil {
					return errors.Wrap(err, "resolving extension path")
				}
				exPackageCfg.Extension.URI = exPath
			}
			relativeBaseDir := ""
			if flags.PackageTomlPath != "" {
				exPackageCfg, err = packageConfigReader.Read(flags.PackageTomlPath)
				if err != nil {
					return errors.Wrap(err, "reading config")
				}

				relativeBaseDir, err = filepath.Abs(filepath.Dir(flags.PackageTomlPath))
				if err != nil {
					return errors.Wrap(err, "getting absolute path for config")
				}
			}
			name := args[0]
			if flags.Format == client.FormatFile {
				switch ext := filepath.Ext(name); ext {
				case client.CNBExtension:
				case "":
					name += client.CNBExtension
				default:
					logger.Warnf("%s is not a valid extension for a packaged extension. Packaged extensions must have a %s extension", style.Symbol(ext), style.Symbol(client.CNBExtension))
				}
			}

			targets, err := processExtensionPackageTargets(flags.Path, packageConfigReader, exPackageCfg)
			if err != nil {
				return err
			}

			daemon := !flags.Publish && flags.Format == ""
			multiArchCfg, err := processMultiArchitectureConfig(logger, flags.Targets, targets, daemon)
			if err != nil {
				return err
			}

			if len(multiArchCfg.Targets()) == 0 {
				logger.Infof("Pro tip: use --target flag OR [[targets]] in buildpack.toml to specify the desired platform (os/arch/variant); using os %s", style.Symbol(exPackageCfg.Platform.OS))
			} else {
				// FIXME: Check if we can copy the config files during layers creation.
				filesToClean, err := multiArchCfg.CopyConfigFiles(exPath, "extension")
				if err != nil {
					return err
				}
				defer clean(filesToClean)
			}

			if err := packager.PackageExtension(cmd.Context(), client.PackageBuildpackOptions{
				RelativeBaseDir: relativeBaseDir,
				Name:            name,
				Format:          flags.Format,
				Config:          exPackageCfg,
				Publish:         flags.Publish,
				PullPolicy:      pullPolicy,
				Targets:         multiArchCfg.Targets(),
			}); err != nil {
				return err
			}

			action := "created"
			location := "docker daemon"
			if flags.Publish {
				action = "published"
				location = "registry"
			}
			if flags.Format == client.FormatFile {
				location = "file"
			}
			logger.Infof("Successfully %s package %s and saved to %s", action, style.Symbol(name), location)
			return nil
		}),
	}

	// flags will be added here
	cmd.Flags().StringVarP(&flags.PackageTomlPath, "config", "c", "", "Path to package TOML config")
	cmd.Flags().StringVarP(&flags.Format, "format", "f", "", `Format to save package as ("image" or "file")`)
	cmd.Flags().BoolVar(&flags.Publish, "publish", false, `Publish the extension directly to the container registry specified in <name>, instead of the daemon (applies to "--format=image" only).`)
	cmd.Flags().StringVar(&flags.Policy, "pull-policy", "", "Pull policy to use. Accepted values are always, never, and if-not-present. The default is always")
	cmd.Flags().StringVarP(&flags.Path, "path", "p", "", "Path to the Extension that needs to be packaged")
	cmd.Flags().StringSliceVarP(&flags.Targets, "target", "t", nil,
		`Target platforms to build for.
Targets should be in the format '[os][/arch][/variant]:[distroname@osversion@anotherversion];[distroname@osversion]'.
- To specify two different architectures: '--target "linux/amd64" --target "linux/arm64"'
- To specify the distribution version: '--target "linux/arm/v6:ubuntu@14.04"'
- To specify multiple distribution versions: '--target "linux/arm/v6:ubuntu@14.04"  --target "linux/arm/v6:ubuntu@16.04"'
	`)
	AddHelpFlag(cmd, "package")
	return cmd
}

func validateExtensionPackageFlags(p *ExtensionPackageFlags) error {
	if p.Publish && p.Policy == image.PullNever.String() {
		return errors.Errorf("--publish and --pull-policy=never cannot be used together. The --publish flag requires the use of remote images.")
	}
	return nil
}

// processExtensionPackageTargets returns the list of targets defined on the extension.toml
func processExtensionPackageTargets(path string, packageConfigReader PackageConfigReader, bpPackageCfg pubbldpkg.Config) ([]dist.Target, error) {
	var targets []dist.Target

	// Read targets from extension.toml
	pathToExtensionToml := filepath.Join(path, "extension.toml")
	if _, err := os.Stat(pathToExtensionToml); err == nil {
		buildpackCfg, err := packageConfigReader.ReadBuildpackDescriptor(pathToExtensionToml)
		if err != nil {
			return nil, err
		}
		targets = buildpackCfg.Targets()
	}

	return targets, nil
}
