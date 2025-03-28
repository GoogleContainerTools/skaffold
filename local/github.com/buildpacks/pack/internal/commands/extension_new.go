package commands

import (
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
)

// ExtensionNewFlags define flags provided to the ExtensionNew command
type ExtensionNewFlags struct {
	API     string
	Path    string
	Stacks  []string
	Version string
}

// extensioncreator type to be added here and argument also to be added in the function

// ExtensionNew generates the scaffolding of an extension
func ExtensionNew(logger logging.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new <id>",
		Short:   "Creates basic scaffolding of an extension",
		Args:    cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Example: "pack extension new <example-extension>",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			// logic will go here
			return nil
		}),
	}

	// flags will go here

	AddHelpFlag(cmd, "new")
	return cmd
}
