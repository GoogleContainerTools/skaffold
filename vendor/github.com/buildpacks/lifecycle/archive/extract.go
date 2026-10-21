package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// ErrEscapesRoot reports a tar entry rejected because its path resolves outside the
// destination root. Callers restoring optional data degrade on it instead of failing.
var ErrEscapesRoot = errors.New("path escapes destination root")

type PathMode struct {
	Path string
	Mode os.FileMode
}

var (
	umaskLock      sync.Mutex
	extractCounter int
	originalUmask  int
)

func setUmaskIfNeeded() {
	umaskLock.Lock()
	defer umaskLock.Unlock()
	extractCounter++
	if extractCounter == 1 {
		originalUmask = setUmask(0)
	}
}

func unsetUmaskIfNeeded() {
	umaskLock.Lock()
	defer umaskLock.Unlock()
	extractCounter--
	if extractCounter == 0 {
		_ = setUmask(originalUmask)
	}
}

// Extract reads all entries from TarReader and extracts them to the filesystem.
//
// destRoot must be an existing directory. Every entry is created through an *os.Root
// handle on it, which resolves each path component beneath the root and so cannot be
// redirected by a symlink, whether the tar creates it or it was already on disk.
func Extract(tr TarReader, destRoot string) error {
	setUmaskIfNeeded()
	defer unsetUmaskIfNeeded()

	destRoot = filepath.Clean(destRoot)
	root, err := os.OpenRoot(destRoot)
	if err != nil {
		return errors.Wrapf(err, "failed to open destination root %q", destRoot)
	}
	defer func() { _ = root.Close() }()

	buf := make([]byte, 32*32*1024)
	dirsFound := make(map[string]bool)

	var pathModes, ancestorModes []PathMode
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			for _, pathMode := range ancestorModes { // directories that are newly created and for which there is a header in the tar should have the right permissions
				if err := os.Chmod(pathMode.Path, pathMode.Mode); err != nil {
					return err
				}
			}
			for _, pathMode := range pathModes {
				if err := root.Chmod(pathMode.Path, pathMode.Mode); err != nil {
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
			rel, err := pathRelativeToRoot(hdr.Name, destRoot, true)
			if err != nil {
				return errors.Wrapf(err, "refusing to extract directory %q", hdr.Name)
			}
			if rel == "" {
				if _, err := os.Stat(hdr.Name); os.IsNotExist(err) {
					ancestorModes = append(ancestorModes, PathMode{hdr.Name, hdr.FileInfo().Mode()})
				}
				if err := os.MkdirAll(hdr.Name, os.ModePerm); err != nil { //nolint:gosec // permissions restored from tar headers
					return errors.Wrapf(err, "failed to create directory %q", hdr.Name)
				}
				continue
			}
			if _, err := root.Lstat(rel); os.IsNotExist(err) {
				pathModes = append(pathModes, PathMode{rel, hdr.FileInfo().Mode()})
			}
			if err := root.MkdirAll(rel, os.ModePerm); err != nil {
				return errors.Wrapf(err, "failed to create directory %q", hdr.Name)
			}
			dirsFound[rel] = true

		case tar.TypeReg:
			rel, err := pathRelativeToRoot(hdr.Name, destRoot, false)
			if err != nil {
				return errors.Wrapf(err, "refusing to extract file %q", hdr.Name)
			}
			dirRel := filepath.Dir(rel)
			if !dirsFound[dirRel] {
				if _, err := root.Lstat(dirRel); os.IsNotExist(err) {
					if err := root.MkdirAll(dirRel, applyUmask(os.ModePerm, originalUmask)); err != nil { // if there is no header for the parent directory in the tar, apply the provided umask
						return errors.Wrapf(err, "failed to create parent dir %q for file %q", dirRel, hdr.Name)
					}
					dirsFound[dirRel] = true
				}
			}

			if err := writeFile(root, tr, rel, hdr.FileInfo().Mode(), buf); err != nil {
				return errors.Wrapf(err, "failed to write file %q", hdr.Name)
			}
		case tar.TypeSymlink:
			rel, err := pathRelativeToRoot(hdr.Name, destRoot, false)
			if err != nil {
				return errors.Wrapf(err, "refusing to create symlink %q", hdr.Name)
			}
			if err := createSymlink(root, rel, hdr); err != nil {
				return errors.Wrapf(err, "failed to create symlink %q with target %q", hdr.Name, hdr.Linkname)
			}
		case tar.TypeXGlobalHeader:
			// ignore PAX Global Extended Headers
			continue
		default:
			return fmt.Errorf("unknown file type in tar %d", hdr.Typeflag)
		}
	}
}

func applyUmask(mode os.FileMode, umask int) os.FileMode {
	return os.FileMode(int(mode) &^ umask)
}

func writeFile(root *os.Root, in io.Reader, rel string, mode os.FileMode, buf []byte) (err error) {
	fh, err := root.OpenFile(rel, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
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

// pathRelativeToRoot converts a tar entry path, already joined with destRoot, to a path
// relative to destRoot. An empty result means the entry is a strict ancestor of destRoot:
// legitimate only for the parent TypeDir entries layers.DirLayer emits, which are chmod'd
// rather than created, so the caller handles them outside the root handle.
func pathRelativeToRoot(path, destRoot string, isDir bool) (string, error) {
	rel, err := filepath.Rel(destRoot, path)
	if err != nil {
		return "", fmt.Errorf("failed to determine path %q relative to destination root %q: %w", path, destRoot, err)
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		if isDir {
			cleanPath := filepath.Clean(path) + string(filepath.Separator)
			if strings.HasPrefix(destRoot+string(filepath.Separator), cleanPath) {
				return "", nil
			}
		}
		return "", fmt.Errorf("%w: %q is not under %q", ErrEscapesRoot, path, destRoot)
	}

	return rel, nil
}
