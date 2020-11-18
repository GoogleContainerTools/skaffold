package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type PathMode struct {
	Path string
	Mode os.FileMode
}

// Extract reads all entries from TarReader and extracts them to the filesystem
func Extract(tr TarReader) error {
	// Avoid umask from changing the file permissions in the tar file.
	umask := setUmask(0)
	defer setUmask(umask)

	buf := make([]byte, 32*32*1024)
	dirsFound := make(map[string]bool)

	var pathModes []PathMode
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			for _, pathMode := range pathModes {
				if err := os.Chmod(pathMode.Path, pathMode.Mode); err != nil {
					return err
				}
			}
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "error extracting from archive")
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(hdr.Name); os.IsNotExist(err) {
				pathMode := PathMode{hdr.Name, hdr.FileInfo().Mode()}
				pathModes = append(pathModes, pathMode)
			}
			if err := os.MkdirAll(hdr.Name, os.ModePerm); err != nil {
				return errors.Wrapf(err, "failed to create directory %q", hdr.Name)
			}
			dirsFound[hdr.Name] = true

		case tar.TypeReg, tar.TypeRegA:
			dirPath := filepath.Dir(hdr.Name)
			if !dirsFound[dirPath] {
				if _, err := os.Stat(dirPath); os.IsNotExist(err) {
					if err := os.MkdirAll(dirPath, applyUmask(os.ModePerm, umask)); err != nil {
						return errors.Wrapf(err, "failed to create parent dir %q for file %q", dirPath, hdr.Name)
					}
					dirsFound[dirPath] = true
				}
			}

			if err := writeFile(tr, hdr.Name, hdr.FileInfo().Mode(), buf); err != nil {
				return errors.Wrapf(err, "failed to write file %q", hdr.Name)
			}
		case tar.TypeSymlink:
			if err := createSymlink(hdr); err != nil {
				return errors.Wrapf(err, "failed to create symlink %q with target %q", hdr.Name, hdr.Linkname)
			}
		default:
			return fmt.Errorf("unknown file type in tar %d", hdr.Typeflag)
		}
	}
}

func applyUmask(mode os.FileMode, umask int) os.FileMode {
	return os.FileMode(int(mode) &^ umask)
}

func writeFile(in io.Reader, path string, mode os.FileMode, buf []byte) (err error) {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fh.Close(); err == nil {
			err = closeErr
		}
	}()
	_, err = io.CopyBuffer(fh, in, buf)
	return err
}
