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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

type headerModifier func(*tar.Header)

func CreateMappedTar(w io.Writer, root string, pathMap map[string][]string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for src, dsts := range pathMap {
		for _, dst := range dsts {
			if err := addFileToTar(root, src, dst, tw, nil); err != nil {
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
		if err := addFileToTar(root, path, "", tw, nil); err != nil {
			return err
		}
	}

	return nil
}

func CreateTarWithParents(w io.Writer, root string, paths []string, uid, gid int, modTime time.Time) error {
	headerModifier := func(header *tar.Header) {
		header.ModTime = modTime
		header.Uid = uid
		header.Gid = gid
		header.Uname = ""
		header.Gname = ""
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	// Make sure parent folders are added before files
	// TODO(dgageot): this should probably also be done in CreateTar
	// but I'd rather not break things that people didn't complain about!
	added := map[string]bool{}

	for _, path := range paths {
		var parentsFirst []string
		for p := path; p != "." && !added[p]; p = filepath.Dir(p) {
			parentsFirst = append(parentsFirst, p)
			added[p] = true
		}

		for i := len(parentsFirst) - 1; i >= 0; i-- {
			if err := addFileToTar(root, parentsFirst[i], "", tw, headerModifier); err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateTarGz(w io.Writer, root string, paths []string) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	return CreateTar(gw, root, paths)
}

func addFileToTar(root string, src string, dst string, tw *tar.Writer, hm headerModifier) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}

	mode := fi.Mode()
	if mode&os.ModeSocket != 0 {
		return nil
	}

	var header *tar.Header
	if mode&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}

		if filepath.IsAbs(target) {
			logrus.Warnf("Skipping %s. Only relative symlinks are supported.", src)
			return nil
		}

		header, err = tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
	} else {
		header, err = tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
	}

	if dst == "" {
		tarPath, err := filepath.Rel(root, src)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(tarPath)
	} else {
		header.Name = filepath.ToSlash(dst)
	}

	// Code copied from https://github.com/moby/moby/blob/master/pkg/archive/archive_windows.go
	if runtime.GOOS == "windows" {
		header.Mode = int64(chmodTarEntry(os.FileMode(header.Mode)))
	}
	if hm != nil {
		hm(header)
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if mode.IsRegular() {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("writing real file %q: %w", src, err)
		}
	}

	return nil
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
