package archive

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

// PathInfo associates a path with an os.FileInfo
type PathInfo struct {
	Path string
	Info os.FileInfo
}

// AddFilesToArchive writes entries describing all files to the provided TarWriter
func AddFilesToArchive(tw TarWriter, files []PathInfo) error {
	for _, file := range files {
		if err := AddFileToArchive(tw, file.Path, file.Info); err != nil {
			return err
		}
	}
	return nil
}

// AddFileToArchive writes an entry describing the file at path with the given os.FileInfo to the provided TarWriter
func AddFileToArchive(tw TarWriter, path string, fi os.FileInfo) error {
	if fi.Mode()&os.ModeSocket != 0 {
		return nil
	}
	header, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	header.Name = path

	if fi.Mode()&os.ModeSymlink != 0 {
		var err error
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		header.Linkname = target
	}
	addSysAttributes(header, fi)
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if fi.Mode().IsRegular() {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
	}
	return nil
}

// AddDirToArchive walks dir writes entries describing dir and all of its children files to the provided TarWriter
func AddDirToArchive(tw TarWriter, dir string) error {
	dir = filepath.Clean(dir)

	return filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return AddFileToArchive(tw, file, fi)
	})
}
