package commands

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	bldr "github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

func ConfigTrustedBuilder(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trusted-builders",
		Short: "List, add and remove trusted builders",
		Long: "When pack considers a builder to be trusted, `pack build` operations will use a single lifecycle binary " +
			"called the creator. This is more efficient than using an untrusted builder, where pack will execute " +
			"five separate lifecycle binaries, each in its own container: analyze, detect, restore, build and export.\n\n" +
			"For more on trusted builders, and when to trust or untrust a builder, " +
			"check out our docs here: https://buildpacks.io/docs/tools/pack/concepts/trusted_builders/",
		Aliases: []string{"trusted-builder", "trust-builder", "trust-builders"},
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			listTrustedBuilders(args, logger, cfg)
			return nil
		}),
	}

	listCmd := generateListCmd("trusted-builders", logger, cfg, listTrustedBuilders)
	listCmd.Long = "List Trusted Builders.\n\nShow the builders that are either trusted by default or have been explicitly trusted locally using `trusted-builder add`"
	listCmd.Example = "pack config trusted-builders list"
	cmd.AddCommand(listCmd)

	addCmd := generateAdd("trusted-builders", logger, cfg, cfgPath, addTrustedBuilder)
	addCmd.Long = "Trust builder.\n\nWhen building with this builder, all lifecycle phases will be run in a single container using the builder image."
	addCmd.Example = "pack config trusted-builders add cnbs/sample-stack-run:bionic"
	cmd.AddCommand(addCmd)

	rmCmd := generateRemove("trusted-builders", logger, cfg, cfgPath, removeTrustedBuilder)
	rmCmd.Long = "Stop trusting builder.\n\nWhen building with this builder, all lifecycle phases will be no longer be run in a single container using the builder image."
	rmCmd.Example = "pack config trusted-builders remove cnbs/sample-stack-run:bionic"
	cmd.AddCommand(rmCmd)

	AddHelpFlag(cmd, "trusted-builders")
	return cmd
}

func addTrustedBuilder(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	imageName := args[0]
	builderToTrust := config.TrustedBuilder{Name: imageName}

	isTrusted, err := bldr.IsTrustedBuilder(cfg, imageName)
	if err != nil {
		return err
	}
	if isTrusted || bldr.IsKnownTrustedBuilder(imageName) {
		logger.Infof("Builder %s is already trusted", style.Symbol(imageName))
		return nil
	}

	cfg.TrustedBuilders = append(cfg.TrustedBuilders, builderToTrust)
	if err := config.Write(cfg, cfgPath); err != nil {
		return errors.Wrap(err, "writing config")
	}
	logger.Infof("Builder %s is now trusted", style.Symbol(imageName))

	return nil
}

func removeTrustedBuilder(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	builder := args[0]

	existingTrustedBuilders := cfg.TrustedBuilders
	cfg.TrustedBuilders = []config.TrustedBuilder{}
	for _, trustedBuilder := range existingTrustedBuilders {
		if trustedBuilder.Name == builder {
			continue
		}

		cfg.TrustedBuilders = append(cfg.TrustedBuilders, trustedBuilder)
	}

	// Builder is not in the trusted builder list
	if len(existingTrustedBuilders) == len(cfg.TrustedBuilders) {
		if bldr.IsKnownTrustedBuilder(builder) {
			// Attempted to untrust a known trusted builder
			return errors.Errorf("Builder %s is a known trusted builder. Currently pack doesn't support making these builders untrusted", style.Symbol(builder))
		}

		logger.Infof("Builder %s wasn't trusted", style.Symbol(builder))
		return nil
	}

	err := config.Write(cfg, cfgPath)
	if err != nil {
		return errors.Wrap(err, "writing config file")
	}

	logger.Infof("Builder %s is no longer trusted", style.Symbol(builder))
	return nil
}

func getTrustedBuilders(cfg config.Config) []string {
	var trustedBuilders []string
	for _, knownBuilder := range bldr.KnownBuilders {
		if knownBuilder.Trusted {
			trustedBuilders = append(trustedBuilders, knownBuilder.Image)
		}
	}

	for _, builder := range cfg.TrustedBuilders {
		trustedBuilders = append(trustedBuilders, builder.Name)
	}

	sort.Strings(trustedBuilders)
	return trustedBuilders
}

func listTrustedBuilders(args []string, logger logging.Logger, cfg config.Config) {
	logger.Info("Trusted Builders:")

	trustedBuilders := getTrustedBuilders(cfg)
	for _, builder := range trustedBuilders {
		logger.Infof("  %s", builder)
	}
}
