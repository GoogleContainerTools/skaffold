package archive

import (
	"archive/tar"
	"os"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
)

const (
	symbolicLinkFlagAllowUnprivilegedCreate = 0x2

	// MSWINDOWS pax vendor extensions
	hdrMSWindowsPrefix = "MSWINDOWS."
	hdrFileAttributes  = hdrMSWindowsPrefix + "fileattr"
)

func setUmask(newMask int) (oldMask int) {
	// Not implemented on Windows
	return 0
}

// createSymlink uses the file attributes in the PAX records to decide whether to make a directory or file type symlink.
// We must use the syscall because we often create symlinks when the target does not exist and os.Symlink uses the mode
//   of the target to create the appropriate type of symlink on windows https://github.com/golang/go/issues/39183
func createSymlink(hdr *tar.Header) error {
	var flags uint32 = symbolicLinkFlagAllowUnprivilegedCreate
	if attrStr, ok := hdr.PAXRecords[hdrFileAttributes]; ok {
		attr, err := strconv.ParseUint(attrStr, 10, 32)
		if err != nil {
			return errors.Wrapf(err, "failed to parse file attributes for header %q", hdr.Name)
		}
		if attr&syscall.FILE_ATTRIBUTE_DIRECTORY != 0 {
			flags |= syscall.SYMBOLIC_LINK_FLAG_DIRECTORY
		}
	}

	name, err := syscall.UTF16PtrFromString(hdr.Name)
	if err != nil {
		return err
	}
	target, err := syscall.UTF16PtrFromString(hdr.Linkname)
	if err != nil {
		return err
	}
	return syscall.CreateSymbolicLink(name, target, flags)
}

// addSysAttributes adds PAXRecords containing file attributes
func addSysAttributes(hdr *tar.Header, fi os.FileInfo) {
	attrs := fi.Sys().(*syscall.Win32FileAttributeData).FileAttributes
	hdr.PAXRecords = map[string]string{}
	hdr.PAXRecords[hdrFileAttributes] = strconv.FormatUint(uint64(attrs), 10)
}
