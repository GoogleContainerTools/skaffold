/*
Copyright 2018 The Skaffold Authors

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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func CreateTar(w io.Writer, root string, paths []string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, p := range paths {
		tarPath := filepath.ToSlash(p)
		p := filepath.Join(root, p)
		if err := addFileToTar(p, tarPath, tw); err != nil {
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

func addFileToTar(p string, tarPath string, tw *tar.Writer) error {
	fi, err := os.Lstat(p)
	if err != nil {
		return err
	}
	switch mode := fi.Mode(); {
	case mode.IsRegular():
		tarHeader, err := tar.FileInfoHeader(fi, tarPath)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath

		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return errors.Wrapf(err, "writing real file %s", p)
		}
	case (mode & os.ModeSymlink) != 0:
		target, err := os.Readlink(p)
		if err != nil {
			return err
		}
		if filepath.IsAbs(target) {
			logrus.Warnf("Skipping %s. Only relative symlinks are supported.", p)
			return nil
		}

		tarHeader, err := tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath
		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}
	default:
		logrus.Warnf("Adding possibly unsupported file %s of type %s.", p, mode)
		// Try to add it anyway?
		tarHeader, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}
	}
	return nil
}
