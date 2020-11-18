package launch

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	bashCommandWithScript = `exec bash -c "$@"` // for processes w/o arguments
)

type BashShell struct {
	Exec ExecFunc
}

// Launch launches the given ShellProcess with Bash
//
// It shall execute a Bash command that sources profile scripts and then executes the process in a nested Bash command
// When ShellProcess.Script is true nested Bash script shall be proc.Command with proc.Args provided as argument to Bash
// When ShellProcess.Script is false a Bash command shall be contructed from proc.Command and proc.Args
func (b *BashShell) Launch(proc ShellProcess) error {
	launcher := ""
	for _, profile := range proc.Profiles {
		launcher += fmt.Sprintf("source \"%s\"\n", profile)
	}
	var bashCommand string
	if proc.Script {
		bashCommand = bashCommandWithScript
	} else {
		bashCommand = bashCommandWithTokens(len(proc.Args) + 1)
	}
	launcher += bashCommand
	if err := b.Exec("/bin/bash", append([]string{
		"bash", "-c",
		launcher, proc.Caller, proc.Command,
	}, proc.Args...), proc.Env); err != nil {
		return errors.Wrap(err, "bash exec")
	}
	return nil
}

// bashCommandWithTokens returns a bash script that should be executed with nTokens number of bash arguments
//  Each argument to bash is evaluated by the shell before becoming a token in the resulting script
//  Example:
//    Given nTokens=2 the returned script will contain `"$(eval echo \"$0\")" "$(eval echo \"$1\")"`
//      and should be evaluated with  `bash -c '"$(eval echo \"$0\")" "$(eval echo \"$1\")"' <command> <arg>'
//    Token evaluation example:
//      "$(eval echo \"$0\"`)" //  given $0='$FOO' and $FOO='arg with spaces" && quotes'
//      -> "$(eval echo \"'$FOO'\")"
//      -> "$(echo \"'arg with spaces" && quotes'\")"
//      -> "arg with spaces\" && quotes" // this is an evaluated and properly quoted token
func bashCommandWithTokens(nTokens int) string {
	commandScript := `"$(eval echo \"$0\")"`
	for i := 1; i < nTokens; i++ {
		commandScript += fmt.Sprintf(` "$(eval echo \"${%d}\")"`, i)
	}
	return fmt.Sprintf(`exec bash -c '%s' "${@:1}"`, commandScript)
}
