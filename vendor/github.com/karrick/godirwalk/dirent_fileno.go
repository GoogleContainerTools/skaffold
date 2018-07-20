// +build dragonfly freebsd openbsd netbsd

package godirwalk

import "syscall"

func direntIno(de *syscall.Dirent) uint64 {
	return uint64(de.Fileno)
}
