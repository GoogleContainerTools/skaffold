//go:build unix

package archive

import (
	"io/fs"
	"syscall"
)

func init() {
	Umask = fs.FileMode(syscall.Umask(0))
	syscall.Umask(int(Umask))
}
