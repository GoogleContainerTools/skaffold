package commands

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
)

type CompletionFlags struct {
	Shell string
}

type completionFunc func(packHome string, cmd *cobra.Command) (path string, err error)

var shellExtensions = map[string]completionFunc{
	"bash": func(packHome string, cmd *cobra.Command) (path string, err error) {
		p := filepath.Join(packHome, "completion.sh")
		return p, cmd.GenBashCompletionFile(p)
	},
	"fish": func(packHome string, cmd *cobra.Command) (path string, err error) {
		p := filepath.Join(packHome, "completion.fish")
		return p, cmd.GenFishCompletionFile(p, true)
	},
	"powershell": func(packHome string, cmd *cobra.Command) (path string, err error) {
		p := filepath.Join(packHome, "completion.ps1")
		return p, cmd.GenPowerShellCompletionFile(p)
	},
	"zsh": func(packHome string, cmd *cobra.Command) (path string, err error) {
		p := filepath.Join(packHome, "completion.zsh")
		return p, cmd.GenZshCompletionFile(p)
	},
}

func CompletionCommand(logger logging.Logger, packHome string) *cobra.Command {
	var flags CompletionFlags
	var completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "Outputs completion script location",
		Long: `Generates completion script and outputs its location.

To configure your bash shell to load completions for each session, add the following to your '.bashrc' or '.bash_profile':

	. $(pack completion)

To configure your fish shell to load completions for each session, add the following to your '~/.config/fish/config.fish':

	source (pack completion --shell fish)

To configure your powershell to load completions for each session, add the following to your '$Home\[My ]Documents\PowerShell\
Microsoft.PowerShell_profile.ps1':

	. $(pack completion --shell powershell)

To configure your zsh shell to load completions for each session, add the following to your '.zshrc':

	. $(pack completion --shell zsh)

  
	`,
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			completionFunc, ok := shellExtensions[flags.Shell]
			if !ok {
				return errors.Errorf("%s is unsupported shell", flags.Shell)
			}

			if err := os.MkdirAll(packHome, os.ModePerm); err != nil {
				return err
			}

			completionPath, err := completionFunc(packHome, cmd.Parent())
			if err != nil {
				return err
			}

			logger.Info(completionPath)
			return nil
		}),
	}

	completionCmd.Flags().StringVarP(&flags.Shell, "shell", "s", "bash", "Generates completion file for [bash|fish|powershell|zsh]")
	return completionCmd
}
