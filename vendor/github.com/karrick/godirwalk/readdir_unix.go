// +build !windows

package godirwalk

import (
	"os"
	"syscall"
	"unsafe"
)

func readdirents(osDirname string, scratchBuffer []byte) (Dirents, error) {
	dh, err := os.Open(osDirname)
	if err != nil {
		return nil, err
	}
	fd := int(dh.Fd())

	if len(scratchBuffer) < MinimumScratchBufferSize {
		scratchBuffer = make([]byte, DefaultScratchBufferSize)
	}

	var entries Dirents
	var de *syscall.Dirent

	for {
		n, err := syscall.ReadDirent(fd, scratchBuffer)
		if err != nil {
			_ = dh.Close() // ignore potential error returned by Close
			return nil, err
		}
		if n <= 0 {
			break // end of directory reached
		}
		// Loop over the bytes returned by reading the directory entries.
		buf := scratchBuffer[:n]
		for len(buf) > 0 {
			de = (*syscall.Dirent)(unsafe.Pointer(&buf[0])) // point entry to first syscall.Dirent in buffer
			buf = buf[de.Reclen:]                           // advance buffer for next iteration through loop

			if inoFromDirent(de) == 0 {
				continue // this item has been deleted, but its entry not yet removed from directory listing
			}

			nameSlice := nameFromDirent(de)
			namlen := len(nameSlice)
			if (namlen == 0) || (namlen == 1 && nameSlice[0] == '.') || (namlen == 2 && nameSlice[0] == '.' && nameSlice[1] == '.') {
				continue // skip unimportant entries
			}
			osChildname := string(nameSlice)

			mode, err := modeType(de, osDirname, osChildname)
			if err != nil {
				_ = dh.Close() // ignore potential error returned by Close
				return nil, err
			}

			entries = append(entries, &Dirent{name: osChildname, modeType: mode})
		}
	}

	if err = dh.Close(); err != nil {
		return nil, err
	}
	return entries, nil
}

func readdirnames(osDirname string, scratchBuffer []byte) ([]string, error) {
	dh, err := os.Open(osDirname)
	if err != nil {
		return nil, err
	}
	fd := int(dh.Fd())

	if len(scratchBuffer) < MinimumScratchBufferSize {
		scratchBuffer = make([]byte, DefaultScratchBufferSize)
	}

	var entries []string
	var de *syscall.Dirent

	for {
		n, err := syscall.ReadDirent(fd, scratchBuffer)
		if err != nil {
			_ = dh.Close() // ignore potential error returned by Close
			return nil, err
		}
		if n <= 0 {
			break // end of directory reached
		}
		// Loop over the bytes returned by reading the directory entries.
		buf := scratchBuffer[:n]
		for len(buf) > 0 {
			de = (*syscall.Dirent)(unsafe.Pointer(&buf[0])) // point entry to first syscall.Dirent in buffer
			buf = buf[de.Reclen:]                           // advance buffer for next iteration through loop

			if inoFromDirent(de) == 0 {
				continue // this item has been deleted, but its entry not yet removed from directory listing
			}

			nameSlice := nameFromDirent(de)
			namlen := len(nameSlice)
			if (namlen == 0) || (namlen == 1 && nameSlice[0] == '.') || (namlen == 2 && nameSlice[0] == '.' && nameSlice[1] == '.') {
				continue // skip unimportant entries
			}
			osChildname := string(nameSlice)

			entries = append(entries, osChildname)
		}
	}

	if err = dh.Close(); err != nil {
		return nil, err
	}
	return entries, nil
}
