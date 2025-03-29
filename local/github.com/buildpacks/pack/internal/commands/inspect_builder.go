package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder/writer"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use builder inspect instead.
func InspectBuilder(
	logger logging.Logger,
	cfg config.Config,
	inspector BuilderInspector,
	writerFactory writer.BuilderWriterFactory,
) *cobra.Command {
	var flags BuilderInspectFlags
	cmd := &cobra.Command{
		Use:     "inspect-builder <builder-image-name>",
		Args:    cobra.MaximumNArgs(2),
		Hidden:  true,
		Short:   "Show information about a builder",
		Example: "pack inspect-builder cnbs/sample-builder:bionic",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			imageName := cfg.DefaultBuilder
			if len(args) >= 1 {
				imageName = args[0]
			}

			if imageName == "" {
				suggestSettingBuilder(logger, inspector)
				return client.NewSoftError()
			}

			return inspectBuilder(logger, imageName, flags, cfg, inspector, writerFactory)
		}),
	}
	cmd.Flags().IntVarP(&flags.Depth, "depth", "d", builder.OrderDetectionMaxDepth, "Max depth to display for Detection Order.\nOmission of this flag or values < 0 will display the entire tree.")
	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "human-readable", "Output format to display builder detail (json, yaml, toml, human-readable).\nOmission of this flag will display as human-readable.")
	AddHelpFlag(cmd, "inspect-builder")
	return cmd
}
