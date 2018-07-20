// +build darwin linux

package godirwalk

import "syscall"

func direntIno(de *syscall.Dirent) uint64 {
	return de.Ino
}
