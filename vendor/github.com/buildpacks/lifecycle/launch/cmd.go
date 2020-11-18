package launch

import (
	"github.com/pkg/errors"
)

type CmdShell struct {
	Exec ExecFunc
}

// Launch launches the given ShellProcess with cmd
func (c *CmdShell) Launch(proc ShellProcess) error {
	var commandTokens []string
	for _, profile := range proc.Profiles {
		commandTokens = append(commandTokens, "call", profile, "&&")
	}
	commandTokens = append(commandTokens, proc.Command)
	commandTokens = append(commandTokens, proc.Args...)
	if err := c.Exec("cmd",
		append([]string{"cmd", "/q", "/v:on", "/s", "/c"}, commandTokens...), proc.Env,
	); err != nil {
		return errors.Wrap(err, "cmd execute")
	}
	return nil
}
