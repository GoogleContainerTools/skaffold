//go:build !windows && !plan9

package copy

import (
	"io/fs"
	"os"
	"syscall"
)

func preserveOwner(src, dest string, info fs.FileInfo) (err error) {
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
