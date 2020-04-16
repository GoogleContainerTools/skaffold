// +build linux darwin

package archive

import (
	"golang.org/x/sys/unix"
)

func setUmask(newMask int) (oldMask int) {
	return unix.Umask(newMask)
}
