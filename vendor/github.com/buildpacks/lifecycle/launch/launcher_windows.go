package launch

import (
	"os"
	"os/exec"
)

const (
	CNBDir     = `c:\cnb`
	exe        = ".exe"
	appProfile = ".profile.bat"
)

var (
	DefaultShell = &CmdShell{Exec: OSExecFunc}
)

func OSExecFunc(argv0 string, argv []string, envv []string) error {
	c := exec.Command(argv[0], argv[1:]...) // #nosec G204
	c.Env = envv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
