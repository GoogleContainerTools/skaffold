//go:build !windows && !plan9
// +build !windows,!plan9

package copy

import (
	"os"
	"syscall"
)

func preserveOwner(src, dest string, info fileInfo) (err error) {
	if info == nil {
		if info, err = os.Stat(src); err != nil {
			return err
		}
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if err := os.Chown(dest, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}
	}
	return nil
}
