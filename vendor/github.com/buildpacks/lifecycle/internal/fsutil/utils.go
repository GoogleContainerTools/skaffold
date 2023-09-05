package fsutil

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func Copy(src, dst string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}

	switch {
	case fi.Mode().IsDir():
		if err := copyDir(src, dst); err != nil {
			return err
		}
	case fi.Mode().IsRegular():
		if err := copyFile(src, dst); err != nil {
			return err
		}
	case fi.Mode()&os.ModeSymlink != 0:
		if err := copySymlink(src, dst); err != nil {
			return err
		}
	default:
		// ignore edge cases (unix socket, named pipe, etc.)
	}
	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	children, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, child := range children {
		srcPath := filepath.Join(src, child.Name())
		dstPath := filepath.Join(dst, child.Name())
		if err := Copy(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

// FilesWithExtensions returns a list of all files in directory that end in any of the extensions provided.
// top level only - does not recursively visit directories.
func FilesWithExtensions(dir string, extensions []string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	matches := []string{}
	for _, f := range files {
		if !f.IsDir() {
			for _, extension := range extensions {
				if strings.HasSuffix(f.Name(), extension) {
					matches = append(matches, filepath.Join(dir, f.Name()))
				}
			}
		}
	}
	return matches, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(target, dst)
}

func RenameWithWindowsFallback(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		switch {
		case runtime.GOOS == "windows":
			// On Windows, when using process isolation, we could encounter https://github.com/moby/moby/issues/38256
			// which causes renames inside mounted volumes to fail for an unknown reason.
			if err = copyDir(src, dst); err != nil {
				return err
			}
			return os.RemoveAll(src)
		default:
			return err
		}
	}
	return nil
}
