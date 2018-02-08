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

func CreateDockerTarContext(dockerfilePath, context string, paths []string, w io.Writer) error {
	// Write everything to memory, then flush to disk at the end.
	// This prevents recursion problems, where the output file can end up
	// in the context itself during creation.
	gw := gzip.NewWriter(w)
	defer gw.Close()

	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		var tarPath string
		if absPath == dockerfilePath {
			tarPath = "Dockerfile"
		} else {
			tarPath, err = filepath.Rel(context, absPath)
			if err != nil {
				return err
			}
		}
		if err := addFileToTar(p, tarPath, tw); err != nil {
			return err
		}

	}
	return nil
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

		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			return errors.Wrapf(err, "writing real file %s", p)
		}
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(p)
		if err != nil {
			return err
		}
		tarHeader, err := tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
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
