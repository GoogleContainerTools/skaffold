package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

var mirrors []string

func ConfigRunImagesMirrors(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run-image-mirrors",
		Short: "List, add and remove run image mirrors",
		Args:  cobra.MaximumNArgs(3),
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			listRunImageMirror(args, logger, cfg)
			return nil
		}),
	}

	listCmd := generateListCmd(cmd.Use, logger, cfg, listRunImageMirror)
	listCmd.Long = "List all run image mirrors. If a run image is provided, it will return "
	listCmd.Use = "list [<run-image>]"
	listCmd.Example = "pack config run-image-mirrors list"
	cmd.AddCommand(listCmd)

	addCmd := generateAdd("mirror for a run image", logger, cfg, cfgPath, addRunImageMirror)
	addCmd.Use = "add <image> [-m <mirror...]"
	addCmd.Long = "Set mirrors to other repositories for a given run image"
	addCmd.Example = "pack config run-image-mirrors add cnbs/sample-stack-run:bionic --mirror index.docker.io/cnbs/sample-stack-run:bionic --mirror gcr.io/cnbs/sample-stack-run:bionic"
	addCmd.Flags().StringSliceVarP(&mirrors, "mirror", "m", nil, "Run image mirror"+stringSliceHelp("mirror"))
	cmd.AddCommand(addCmd)

	rmCmd := generateRemove("mirror for a run image", logger, cfg, cfgPath, removeRunImageMirror)
	rmCmd.Use = "remove <image> [-m <mirror...]"
	rmCmd.Long = "Remove mirrors for a given run image. If specific mirrors are passed, they will be removed. " +
		"If no mirrors are provided, all mirrors for the given run image will be removed from the config."
	rmCmd.Example = "pack config run-image-mirrors remove cnbs/sample-stack-run:bionic"
	rmCmd.Flags().StringSliceVarP(&mirrors, "mirror", "m", nil, "Run image mirror"+stringSliceHelp("mirror"))
	cmd.AddCommand(rmCmd)

	AddHelpFlag(cmd, "run-image-mirrors")
	return cmd
}

func addRunImageMirror(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	runImage := args[0]
	if len(mirrors) == 0 {
		logger.Infof("No run image mirrors were provided.")
		return nil
	}

	newMirrors := mirrors
	for _, image := range cfg.RunImages {
		if image.Image == runImage {
			newMirrors = append(newMirrors, image.Mirrors...)
			break
		}
	}

	cfg = config.SetRunImageMirrors(cfg, runImage, dedupAndSortSlice(newMirrors))
	if err := config.Write(cfg, cfgPath); err != nil {
		return errors.Wrapf(err, "failed to write to %s", cfgPath)
	}

	for _, mirror := range mirrors {
		logger.Infof("Run Image %s configured with mirror %s", style.Symbol(runImage), style.Symbol(mirror))
	}
	return nil
}

func removeRunImageMirror(args []string, logger logging.Logger, cfg config.Config, cfgPath string) error {
	image := args[0]

	idx := -1
	for i, runImage := range cfg.RunImages {
		if runImage.Image == image {
			idx = i
		}
	}

	if idx == -1 || len(cfg.RunImages) == 0 {
		// Run Image wasn't found
		logger.Infof("No run image mirrors have been set for %s", style.Symbol(image))
		return nil
	}

	mirrorsMap := stringset.FromSlice(mirrors)
	var newMirrors []string
	for _, currMirror := range cfg.RunImages[idx].Mirrors {
		if _, ok := mirrorsMap[currMirror]; !ok {
			newMirrors = append(newMirrors, currMirror)
		}
	}

	if len(newMirrors) == 0 || len(mirrors) == 0 {
		lastImageIdx := len(cfg.RunImages) - 1
		cfg.RunImages[idx] = cfg.RunImages[lastImageIdx]
		cfg.RunImages = cfg.RunImages[:lastImageIdx]
	} else {
		cfg = config.SetRunImageMirrors(cfg, image, newMirrors)
	}

	if err := config.Write(cfg, cfgPath); err != nil {
		return errors.Wrapf(err, "failed to write to %s", cfgPath)
	}
	if len(mirrors) == 0 {
		logger.Infof("Removed all run image mirrors for %s", style.Symbol(image))
	} else {
		logger.Infof("Removed mirrors %s for %s", strings.Join(mirrors, ", "), style.Symbol(image))
	}

	return nil
}

func listRunImageMirror(args []string, logger logging.Logger, cfg config.Config) {
	var (
		reqImage string
		found    = false
	)

	if len(args) > 0 {
		reqImage = args[0]
	}

	buf := strings.Builder{}
	buf.WriteString("Run Image Mirrors:\n")
	for _, runImage := range cfg.RunImages {
		if (reqImage != "" && runImage.Image == reqImage) || reqImage == "" {
			found = true
			buf.WriteString(fmt.Sprintf("  %s:\n", style.Symbol(runImage.Image)))
			for _, mirror := range runImage.Mirrors {
				buf.WriteString(fmt.Sprintf("    %s\n", mirror))
			}
		}
	}

	if !found {
		suffix := ""
		if reqImage != "" {
			suffix = fmt.Sprintf("for %s", style.Symbol(reqImage))
		}
		logger.Infof("No run image mirrors have been set %s", suffix)
	} else {
		logger.Info(buf.String())
	}
}

func dedupAndSortSlice(slice []string) []string {
	set := stringset.FromSlice(slice)
	var newSlice []string
	for s := range set {
		newSlice = append(newSlice, s)
	}
	sort.Strings(newSlice)
	return newSlice
}
