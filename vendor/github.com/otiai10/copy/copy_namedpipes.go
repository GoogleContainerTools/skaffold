//go:build !windows && !plan9 && !netbsd && !aix && !illumos && !solaris && !js
// +build !windows,!plan9,!netbsd,!aix,!illumos,!solaris,!js

package copy

import (
	"os"
	"path/filepath"
	"syscall"
)

// pcopy is for just named pipes
func pcopy(dest string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}
	return syscall.Mkfifo(dest, uint32(info.Mode()))
}
