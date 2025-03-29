package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"

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

// BuildpackPackageFlags define flags provided to the BuildpackPackage command
type BuildpackPackageFlags struct {
	PackageTomlPath       string
	Format                string
	Policy                string
	BuildpackRegistry     string
	Path                  string
	FlattenExclude        []string
	Targets               []string
	Label                 map[string]string
	Publish               bool
	Flatten               bool
	AppendImageNameSuffix bool
}

// BuildpackPackager packages buildpacks
type BuildpackPackager interface {
	PackageBuildpack(ctx context.Context, options client.PackageBuildpackOptions) error
}

// PackageConfigReader reads BuildpackPackage configs
type PackageConfigReader interface {
	Read(path string) (pubbldpkg.Config, error)
	ReadBuildpackDescriptor(path string) (dist.BuildpackDescriptor, error)
}

// BuildpackPackage packages (a) buildpack(s) into OCI format, based on a package config
func BuildpackPackage(logger logging.Logger, cfg config.Config, packager BuildpackPackager, packageConfigReader PackageConfigReader) *cobra.Command {
	var flags BuildpackPackageFlags
	cmd := &cobra.Command{
		Use:     "package <name> --config <config-path>",
		Short:   "Package a buildpack in OCI format.",
		Args:    cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Example: "pack buildpack package my-buildpack --config ./package.toml\npack buildpack package my-buildpack.cnb --config ./package.toml --f file",
		Long: "buildpack package allows users to package (a) buildpack(s) into OCI format, which can then to be hosted in " +
			"image repositories or persisted on disk as a '.cnb' file. You can also package a number of buildpacks " +
			"together, to enable easier distribution of a set of buildpacks. " +
			"Packaged buildpacks can be used as inputs to `pack build` (using the `--buildpack` flag), " +
			"and they can be included in the configs used in `pack builder create` and `pack buildpack package`. For more " +
			"on how to package a buildpack, see: https://buildpacks.io/docs/buildpack-author-guide/package-a-buildpack/.",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
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
			bpPackageCfg := pubbldpkg.DefaultConfig()
			var bpPath string
			if flags.Path != "" {
				if bpPath, err = filepath.Abs(flags.Path); err != nil {
					return errors.Wrap(err, "resolving buildpack path")
				}
				bpPackageCfg.Buildpack.URI = bpPath
			}
			relativeBaseDir := ""
			if flags.PackageTomlPath != "" {
				bpPackageCfg, err = packageConfigReader.Read(flags.PackageTomlPath)
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
					logger.Warnf("%s is not a valid extension for a packaged buildpack. Packaged buildpacks must have a %s extension", style.Symbol(ext), style.Symbol(client.CNBExtension))
				}
			}
			if flags.Flatten {
				logger.Warn("Flattening a buildpack package could break the distribution specification. Please use it with caution.")
			}

			targets, isCompositeBP, err := processBuildpackPackageTargets(flags.Path, packageConfigReader, bpPackageCfg)
			if err != nil {
				return err
			}

			daemon := !flags.Publish && flags.Format == ""
			multiArchCfg, err := processMultiArchitectureConfig(logger, flags.Targets, targets, daemon)
			if err != nil {
				return err
			}

			if len(multiArchCfg.Targets()) == 0 {
				if isCompositeBP {
					logger.Infof("Pro tip: use --target flag OR [[targets]] in package.toml to specify the desired platform (os/arch/variant); using os %s", style.Symbol(bpPackageCfg.Platform.OS))
				} else {
					logger.Infof("Pro tip: use --target flag OR [[targets]] in buildpack.toml to specify the desired platform (os/arch/variant); using os %s", style.Symbol(bpPackageCfg.Platform.OS))
				}
			} else if !isCompositeBP {
				// FIXME: Check if we can copy the config files during layers creation.
				filesToClean, err := multiArchCfg.CopyConfigFiles(bpPath, "buildpack")
				if err != nil {
					return err
				}
				defer clean(filesToClean)
			}

			if !flags.Publish && flags.AppendImageNameSuffix {
				logger.Warnf("--append-image-name-suffix will be ignored, use combined with --publish")
			}

			if err := packager.PackageBuildpack(cmd.Context(), client.PackageBuildpackOptions{
				RelativeBaseDir:       relativeBaseDir,
				Name:                  name,
				Format:                flags.Format,
				Config:                bpPackageCfg,
				Publish:               flags.Publish,
				AppendImageNameSuffix: flags.AppendImageNameSuffix && flags.Publish,
				PullPolicy:            pullPolicy,
				Registry:              flags.BuildpackRegistry,
				Flatten:               flags.Flatten,
				FlattenExclude:        flags.FlattenExclude,
				Labels:                flags.Label,
				Targets:               multiArchCfg.Targets(),
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

	cmd.Flags().StringVarP(&flags.PackageTomlPath, "config", "c", "", "Path to package TOML config")
	cmd.Flags().StringVarP(&flags.Format, "format", "f", "", `Format to save package as ("image" or "file")`)
	cmd.Flags().BoolVar(&flags.Publish, "publish", false, `Publish the buildpack directly to the container registry specified in <name>, instead of the daemon (applies to "--format=image" only).`)
	cmd.Flags().BoolVar(&flags.AppendImageNameSuffix, "append-image-name-suffix", false, "When publishing to a registry that doesn't allow overwrite existing tags use this flag to append a [os]-[arch] suffix to package <name>")
	cmd.Flags().StringVar(&flags.Policy, "pull-policy", "", "Pull policy to use. Accepted values are always, never, and if-not-present. The default is always")
	cmd.Flags().StringVarP(&flags.Path, "path", "p", "", "Path to the Buildpack that needs to be packaged")
	cmd.Flags().StringVarP(&flags.BuildpackRegistry, "buildpack-registry", "r", "", "Buildpack Registry name")
	cmd.Flags().BoolVar(&flags.Flatten, "flatten", false, "Flatten the buildpack into a single layer")
	cmd.Flags().StringSliceVarP(&flags.FlattenExclude, "flatten-exclude", "e", nil, "Buildpacks to exclude from flattening, in the form of '<buildpack-id>@<buildpack-version>'")
	cmd.Flags().StringToStringVarP(&flags.Label, "label", "l", nil, "Labels to add to packaged Buildpack, in the form of '<name>=<value>'")
	cmd.Flags().StringSliceVarP(&flags.Targets, "target", "t", nil,
		`Target platforms to build for.
Targets should be in the format '[os][/arch][/variant]:[distroname@osversion@anotherversion];[distroname@osversion]'.
- To specify two different architectures: '--target "linux/amd64" --target "linux/arm64"'
- To specify the distribution version: '--target "linux/arm/v6:ubuntu@14.04"'
- To specify multiple distribution versions: '--target "linux/arm/v6:ubuntu@14.04"  --target "linux/arm/v6:ubuntu@16.04"'
	`)
	if !cfg.Experimental {
		cmd.Flags().MarkHidden("flatten")
		cmd.Flags().MarkHidden("flatten-exclude")
	}
	AddHelpFlag(cmd, "package")
	return cmd
}

func validateBuildpackPackageFlags(cfg config.Config, p *BuildpackPackageFlags) error {
	if p.Publish && p.Policy == image.PullNever.String() {
		return errors.Errorf("--publish and --pull-policy never cannot be used together. The --publish flag requires the use of remote images.")
	}
	if p.PackageTomlPath != "" && p.Path != "" {
		return errors.Errorf("--config and --path cannot be used together. Please specify the relative path to the Buildpack directory in the package config file.")
	}

	if p.Flatten {
		if !cfg.Experimental {
			return client.NewExperimentError("Flattening a buildpack package is currently experimental.")
		}

		if len(p.FlattenExclude) > 0 {
			for _, exclude := range p.FlattenExclude {
				if strings.Count(exclude, "@") != 1 {
					return errors.Errorf("invalid format %s; please use '<buildpack-id>@<buildpack-version>' to exclude buildpack from flattening", exclude)
				}
			}
		}
	}
	return nil
}

// processBuildpackPackageTargets returns the list of targets defined in the configuration file; it could be the buildpack.toml or
// the package.toml if the buildpack is a composite buildpack
func processBuildpackPackageTargets(path string, packageConfigReader PackageConfigReader, bpPackageCfg pubbldpkg.Config) ([]dist.Target, bool, error) {
	var (
		targets       []dist.Target
		order         dist.Order
		isCompositeBP bool
	)

	// Read targets from buildpack.toml
	pathToBuildpackToml := filepath.Join(path, "buildpack.toml")
	if _, err := os.Stat(pathToBuildpackToml); err == nil {
		buildpackCfg, err := packageConfigReader.ReadBuildpackDescriptor(pathToBuildpackToml)
		if err != nil {
			return nil, false, err
		}
		targets = buildpackCfg.Targets()
		order = buildpackCfg.Order()
		isCompositeBP = len(order) > 0
	}

	// When composite buildpack, targets are defined in package.toml - See RFC-0128
	if isCompositeBP {
		targets = bpPackageCfg.Targets
	}
	return targets, isCompositeBP, nil
}

func clean(paths []string) error {
	// we need to clean the buildpack.toml for each place where we copied to
	if len(paths) > 0 {
		for _, path := range paths {
			os.Remove(path)
		}
	}
	return nil
}
