package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/internal/target"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
)

//go:generate mockgen -package testmocks -destination testmocks/mock_pack_client.go github.com/buildpacks/pack/internal/commands PackClient
type PackClient interface {
	InspectBuilder(string, bool, ...client.BuilderInspectionModifier) (*client.BuilderInfo, error)
	InspectImage(string, bool) (*client.ImageInfo, error)
	Rebase(context.Context, client.RebaseOptions) error
	CreateBuilder(context.Context, client.CreateBuilderOptions) error
	NewBuildpack(context.Context, client.NewBuildpackOptions) error
	PackageBuildpack(ctx context.Context, opts client.PackageBuildpackOptions) error
	PackageExtension(ctx context.Context, opts client.PackageBuildpackOptions) error
	Build(context.Context, client.BuildOptions) error
	RegisterBuildpack(context.Context, client.RegisterBuildpackOptions) error
	YankBuildpack(client.YankBuildpackOptions) error
	InspectBuildpack(client.InspectBuildpackOptions) (*client.BuildpackInfo, error)
	InspectExtension(client.InspectExtensionOptions) (*client.ExtensionInfo, error)
	PullBuildpack(context.Context, client.PullBuildpackOptions) error
	DownloadSBOM(name string, options client.DownloadSBOMOptions) error
	CreateManifest(ctx context.Context, opts client.CreateManifestOptions) error
	AnnotateManifest(ctx context.Context, opts client.ManifestAnnotateOptions) error
	AddManifest(ctx context.Context, opts client.ManifestAddOptions) error
	DeleteManifest(name []string) error
	RemoveManifest(name string, images []string) error
	PushManifest(client.PushManifestOptions) error
	InspectManifest(string) error
}

func AddHelpFlag(cmd *cobra.Command, commandName string) {
	cmd.Flags().BoolP("help", "h", false, fmt.Sprintf("Help for '%s'", commandName))
}

func CreateCancellableContext() context.Context {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-signals
		cancel()
	}()

	return ctx
}

func logError(logger logging.Logger, f func(cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		err := f(cmd, args)
		if err != nil {
			if _, isSoftError := errors.Cause(err).(client.SoftError); !isSoftError {
				logger.Error(err.Error())
			}

			if _, isExpError := errors.Cause(err).(client.ExperimentError); isExpError {
				configPath, err := config.DefaultConfigPath()
				if err != nil {
					return err
				}
				enableExperimentalTip(logger, configPath)
			}
			return err
		}
		return nil
	}
}

func enableExperimentalTip(logger logging.Logger, configPath string) {
	logging.Tip(logger, "To enable experimental features, run `pack config experimental true` to add %s to %s.", style.Symbol("experimental = true"), style.Symbol(configPath))
}

func stringArrayHelp(name string) string {
	return fmt.Sprintf("\nRepeat for each %s in order (comma-separated lists not accepted)", name)
}

func stringSliceHelp(name string) string {
	return fmt.Sprintf("\nRepeat for each %s in order, or supply once by comma-separated list", name)
}

func getMirrors(config config.Config) map[string][]string {
	mirrors := map[string][]string{}
	for _, ri := range config.RunImages {
		mirrors[ri.Image] = ri.Mirrors
	}
	return mirrors
}

func deprecationWarning(logger logging.Logger, oldCmd, replacementCmd string) {
	logger.Warnf("Command %s has been deprecated, please use %s instead", style.Symbol("pack "+oldCmd), style.Symbol("pack "+replacementCmd))
}

func parseFormatFlag(value string) (types.MediaType, error) {
	var format types.MediaType
	switch value {
	case "oci":
		format = types.OCIImageIndex
	case "docker":
		format = types.DockerManifestList
	default:
		return format, errors.Errorf("%s invalid media type format", value)
	}
	return format, nil
}

// processMultiArchitectureConfig takes an array of targets with format: [os][/arch][/variant]:[distroname@osversion@anotherversion];[distroname@osversion]
// and a list of targets defined in a configuration file (buildpack.toml or package.toml) and creates a multi-architecture configuration
func processMultiArchitectureConfig(logger logging.Logger, userTargets []string, configTargets []dist.Target, daemon bool) (*buildpack.MultiArchConfig, error) {
	var (
		expectedTargets []dist.Target
		err             error
	)
	if len(userTargets) > 0 {
		if expectedTargets, err = target.ParseTargets(userTargets, logger); err != nil {
			return &buildpack.MultiArchConfig{}, err
		}
		if len(expectedTargets) > 1 && daemon {
			// when we are exporting to daemon, only 1 target is allow
			return &buildpack.MultiArchConfig{}, errors.Errorf("when exporting to daemon only one target is allowed")
		}
	}

	multiArchCfg, err := buildpack.NewMultiArchConfig(configTargets, expectedTargets, logger)
	if err != nil {
		return &buildpack.MultiArchConfig{}, err
	}
	return multiArchCfg, nil
}
