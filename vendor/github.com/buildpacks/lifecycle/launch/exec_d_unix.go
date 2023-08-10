//go:build linux || darwin
// +build linux darwin

package launch

import (
	"os"
	"os/exec"
)

func setHandle(cmd *exec.Cmd, f *os.File) error {
	cmd.ExtraFiles = []*os.File{f}
	return nil
}
