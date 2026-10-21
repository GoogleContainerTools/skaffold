//go:build unix

package archive

import (
	"archive/tar"
	"os"

	"golang.org/x/sys/unix"
)

func setUmask(newMask int) (oldMask int) {
	return unix.Umask(newMask)
}

func createSymlink(root *os.Root, rel string, hdr *tar.Header) error {
	return root.Symlink(hdr.Linkname, rel)
}

func addSysAttributes(hdr *tar.Header, fi os.FileInfo) {
}
