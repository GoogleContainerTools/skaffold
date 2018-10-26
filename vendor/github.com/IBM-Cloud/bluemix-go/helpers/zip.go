package helpers

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

//Unzip src to dest
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}

	for _, f := range r.File {
		err := extractFileInZipArchive(dest, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractFileInZipArchive(dest string, f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(dest, f.Name)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}
	err = os.MkdirAll(filepath.Dir(path), f.Mode())
	if err != nil {
		return err
	}
	zf, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer zf.Close()
	_, err = io.Copy(zf, rc)
	return err
}
