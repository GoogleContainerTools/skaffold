package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version can be set via:
// -ldflags="-X 'github.com/google/go-containerregistry/cmd/crane/cmd.Version=$TAG'"
var Version string

func init() {
	if Version == "" {
		i, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		Version = i.Main.Version
	}
}

// NewCmdVersion creates a new cobra.Command for the version subcommand.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Long: `The version string is completely dependent on how the binary was built, so you should not depend on the version format. It may change without notice.

This could be an arbitrary string, if specified via -ldflags.
This could also be the go module version, if built with go modules (often "(devel)").`,
		Args: cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			if Version == "" {
				fmt.Println("could not determine build information")
			} else {
				fmt.Println(Version)
			}
		},
	}
}
