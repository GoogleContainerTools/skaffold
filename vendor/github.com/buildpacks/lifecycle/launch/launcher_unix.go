//go:build linux || darwin
// +build linux darwin

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
