//go:build linux || darwin
// +build linux darwin

package archive

import (
	"archive/tar"
	"os"

	"golang.org/x/sys/unix"
)

func setUmask(newMask int) (oldMask int) {
	return unix.Umask(newMask)
}

func createSymlink(hdr *tar.Header) error {
	return os.Symlink(hdr.Linkname, hdr.Name)
}

func addSysAttributes(hdr *tar.Header, fi os.FileInfo) {
}
