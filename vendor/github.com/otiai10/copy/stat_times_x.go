//go:build plan9 || netbsd
// +build plan9 netbsd

package copy

import (
	"os"
)

// TODO: check plan9 netbsd in future
func getTimeSpec(info os.FileInfo) timespec {
	times := timespec{
		Mtime: info.ModTime(),
		Atime: info.ModTime(),
		Ctime: info.ModTime(),
	}
	return times
}
