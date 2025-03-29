package commands

import (
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

func ConfigExperimental(logger logging.Logger, cfg config.Config, cfgPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experimental [<true | false>]",
		Args:  cobra.MaximumNArgs(1),
		Short: "List and set the current 'experimental' value from the config",
		Long: "Experimental features in pack are gated, and require you adding setting `experimental=true` to the Pack Config, either manually, or using this command.\n\n" +
			"* Running `pack config experimental` prints whether experimental features are currently enabled.\n" +
			"* Running `pack config experimental <true | false>` enables or disables experimental features.",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) == 0:
				if cfg.Experimental {
					logger.Infof("Experimental features are enabled! To turn them off, run `pack config experimental false`")
				} else {
					logger.Info("Experimental features aren't currently enabled. To enable them, run `pack config experimental true`")
				}
			default:
				val, err := strconv.ParseBool(args[0])
				if err != nil {
					return errors.Wrapf(err, "invalid value %s provided", style.Symbol(args[0]))
				}
				cfg.Experimental = val
				if cfg.Experimental {
					cfg.LayoutRepositoryDir = filepath.Join(filepath.Dir(cfgPath), "layout-repo")
				} else {
					cfg.LayoutRepositoryDir = ""
				}

				if err = config.Write(cfg, cfgPath); err != nil {
					return errors.Wrap(err, "writing to config")
				}

				if cfg.Experimental {
					logger.Info("Experimental features enabled!")
				} else {
					logger.Info("Experimental features disabled")
				}
			}

			return nil
		}),
	}

	AddHelpFlag(cmd, "experimental")
	return cmd
}
