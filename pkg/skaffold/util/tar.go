/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func CreateMappedTar(w io.Writer, root string, pathMap map[string][]string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for src, dsts := range pathMap {
		for _, dst := range dsts {
			if err := addFileToTar(root, src, dst, tw); err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateTar(w io.Writer, root string, paths []string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, path := range paths {
		if err := addFileToTar(root, path, "", tw); err != nil {
			return err
		}
	}

	return nil
}

func CreateTarGz(w io.Writer, root string, paths []string) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	return CreateTar(gw, root, paths)
}

func addFileToTar(root string, src string, dst string, tw *tar.Writer) error {
	var (
		absPath string
		err     error
	)

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	if filepath.IsAbs(src) {
		absPath = src
	} else {
		absPath, err = filepath.Abs(src)
		if err != nil {
			return err
		}
	}

	tarPath := dst
	if tarPath == "" {
		tarPath, err = filepath.Rel(absRoot, absPath)
		if err != nil {
			return err
		}
	}
	tarPath = filepath.ToSlash(tarPath)

	fi, err := os.Lstat(absPath)
	if err != nil {
		return err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		tarHeader, err := tar.FileInfoHeader(fi, tarPath)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath

		if err := writeHeader(tw, tarHeader); err != nil {
			return err
		}
	case mode.IsRegular():
		tarHeader, err := tar.FileInfoHeader(fi, tarPath)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath

		if err := writeHeader(tw, tarHeader); err != nil {
			return err
		}

		f, err := os.Open(absPath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return errors.Wrapf(err, "writing real file %s", absPath)
		}
	case (mode & os.ModeSymlink) != 0:
		target, err := os.Readlink(absPath)
		if err != nil {
			return err
		}
		if filepath.IsAbs(target) {
			logrus.Warnf("Skipping %s. Only relative symlinks are supported.", absPath)
			return nil
		}

		tarHeader, err := tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath
		if err := writeHeader(tw, tarHeader); err != nil {
			return err
		}
	default:
		logrus.Warnf("Adding possibly unsupported file %s of type %s.", absPath, mode)
		// Try to add it anyway?
		tarHeader, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		if err := writeHeader(tw, tarHeader); err != nil {
			return err
		}
	}
	return nil
}

// Code copied from https://github.com/moby/moby/blob/master/pkg/archive/archive_windows.go
func writeHeader(tw *tar.Writer, tarHeader *tar.Header) error {
	if runtime.GOOS == "windows" {
		tarHeader.Mode = int64(chmodTarEntry(os.FileMode(tarHeader.Mode)))
	}

	return tw.WriteHeader(tarHeader)
}

// Code copied from https://github.com/moby/moby/blob/master/pkg/archive/archive_windows.go
func chmodTarEntry(perm os.FileMode) os.FileMode {
	//perm &= 0755 // this 0-ed out tar flags (like link, regular file, directory marker etc.)
	permPart := perm & os.ModePerm
	noPermPart := perm &^ os.ModePerm
	// Add the x bit: make everything +x from windows
	permPart |= 0111
	permPart &= 0755

	return noPermPart | permPart
}
