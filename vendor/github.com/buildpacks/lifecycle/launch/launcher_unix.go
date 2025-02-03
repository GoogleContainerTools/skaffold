//go:build unix

package launch

import "syscall"

const (
	CNBDir     = `/cnb`
	exe        = ""
	appProfile = ".profile"
)

var (
	OSExecFunc   = syscall.Exec
	DefaultShell = &BashShell{Exec: OSExecFunc}
)
