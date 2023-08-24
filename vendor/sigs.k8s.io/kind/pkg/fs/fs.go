/*
Copyright 2018 The Kubernetes Authors.

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

// Package fs contains utilities for interacting with the host filesystem
// in a docker friendly way
// TODO(bentheelder): this should be internal
package fs

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// TempDir is like os.MkdirTemp, but more docker friendly
func TempDir(dir, prefix string) (name string, err error) {
	// create a tempdir as normal
	name, err = os.MkdirTemp(dir, prefix)
	if err != nil {
		return "", err
	}
	// on macOS $TMPDIR is typically /var/..., which is not mountable
	// /private/var/... is the mountable equivalent
	if runtime.GOOS == "darwin" && strings.HasPrefix(name, "/var/") {
		name = filepath.Join("/private", name)
	}
	return name, nil
}

// IsAbs is like filepath.IsAbs but also considering posix absolute paths
// to be absolute even if filepath.IsAbs would not
// This fixes the case of Posix paths on Windows
func IsAbs(hostPath string) bool {
	return path.IsAbs(hostPath) || filepath.IsAbs(hostPath)
}

// Copy recursively directories, symlinks, files copies from src to dst
// Copy will make dirs as necessary, and keep file modes
// Symlinks will be dereferenced similar to `cp -r src dst`
func Copy(src, dst string) error {
	// get source info
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	// make sure dest dir exists
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}
	// do real copy work
	return copy(src, dst, info)
}

func copy(src, dst string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(src, dst)
	}
	if info.IsDir() {
		return copyDir(src, dst, info)
	}
	return copyFile(src, dst, info)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) (err error) {
	// get source information
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return copyFile(src, dst, info)
}

func copyFile(src, dst string, info os.FileInfo) error {
	// open src for reading
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// create dst file
	// this is like f, err := os.Create(dst); os.Chmod(f.Name(), src.Mode())
	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	// make sure we close the file
	defer func() {
		closeErr := out.Close()
		// if we weren't returning an error
		if err == nil {
			err = closeErr
		}
	}()
	// actually copy
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

// copySymlink dereferences and then copies a symlink
func copySymlink(src, dst string) error {
	// read through the symlink
	realSrc, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	info, err := os.Lstat(realSrc)
	if err != nil {
		return err
	}
	// copy the underlying contents
	return copy(realSrc, dst, info)
}

func copyDir(src, dst string, info os.FileInfo) error {
	// make sure the target dir exists
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	// copy every source dir entry
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entrySrc := filepath.Join(src, entry.Name())
		entryDst := filepath.Join(dst, entry.Name())
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if err := copy(entrySrc, entryDst, fileInfo); err != nil {
			return err
		}
	}
	return nil
}
