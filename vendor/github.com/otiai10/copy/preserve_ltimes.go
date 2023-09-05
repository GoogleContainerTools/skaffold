//go:build !windows && !plan9 && !js
// +build !windows,!plan9,!js

package copy

import (
	"golang.org/x/sys/unix"
)

func preserveLtimes(src, dest string) error {
	info := new(unix.Stat_t)
	if err := unix.Lstat(src, info); err != nil {
		return err
	}

	return unix.Lutimes(dest, []unix.Timeval{
		unix.NsecToTimeval(info.Atim.Nano()),
		unix.NsecToTimeval(info.Mtim.Nano()),
	})
}
