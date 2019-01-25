/*
Copyright 2018 Google LLC

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
	"bufio"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/sirupsen/logrus"
)

type WhitelistEntry struct {
	Path            string
	PrefixMatchOnly bool
}

var whitelist = []WhitelistEntry{
	{
		Path:            "/kaniko",
		PrefixMatchOnly: false,
	},
	{
		// /var/run is a special case. It's common to mount in /var/run/docker.sock or something similar
		// which leads to a special mount on the /var/run/docker.sock file itself, but the directory to exist
		// in the image with no way to tell if it came from the base image or not.
		Path:            "/var/run",
		PrefixMatchOnly: false,
	},
	{
		// similarly, we whitelist /etc/mtab, since there is no way to know if the file was mounted or came
		// from the base image
		Path:            "/etc/mtab",
		PrefixMatchOnly: false,
	},
}

// GetFSFromImage extracts the layers of img to root
// It returns a list of all files extracted
func GetFSFromImage(root string, img v1.Image) ([]string, error) {
	whitelist, err := fileSystemWhitelist(constants.WhitelistPath)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Mounted directories: %v", whitelist)
	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}
	extractedFiles := []string{}

	for i, l := range layers {
		logrus.Infof("Extracting layer %d", i)
		r, err := l.Uncompressed()
		if err != nil {
			return nil, err
		}
		tr := tar.NewReader(r)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			path := filepath.Join(root, filepath.Clean(hdr.Name))
			base := filepath.Base(path)
			dir := filepath.Dir(path)
			if strings.HasPrefix(base, ".wh.") {
				logrus.Debugf("Whiting out %s", path)
				name := strings.TrimPrefix(base, ".wh.")
				if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
					return nil, errors.Wrapf(err, "removing whiteout %s", hdr.Name)
				}
				continue
			}
			if err := extractFile(root, hdr, tr); err != nil {
				return nil, err
			}
			extractedFiles = append(extractedFiles, filepath.Join(root, filepath.Clean(hdr.Name)))
		}
	}
	return extractedFiles, nil
}

// DeleteFilesystem deletes the extracted image file system
func DeleteFilesystem() error {
	logrus.Info("Deleting filesystem...")
	return filepath.Walk(constants.RootDir, func(path string, info os.FileInfo, _ error) error {
		whitelisted, err := CheckWhitelist(path)
		if err != nil {
			return err
		}
		if whitelisted || ChildDirInWhitelist(path, constants.RootDir) {
			logrus.Debugf("Not deleting %s, as it's whitelisted", path)
			return nil
		}
		if path == constants.RootDir {
			return nil
		}
		return os.RemoveAll(path)
	})
}

// ChildDirInWhitelist returns true if there is a child file or directory of the path in the whitelist
func ChildDirInWhitelist(path, directory string) bool {
	for _, d := range constants.KanikoBuildFiles {
		dirPath := filepath.Join(directory, d)
		if HasFilepathPrefix(dirPath, path, false) {
			return true
		}
	}
	for _, d := range whitelist {
		dirPath := filepath.Join(directory, d.Path)
		if HasFilepathPrefix(dirPath, path, d.PrefixMatchOnly) {
			return true
		}
	}
	return false
}

// unTar returns a list of files that have been extracted from the tar archive at r to the path at dest
func unTar(r io.Reader, dest string) ([]string, error) {
	var extractedFiles []string
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := extractFile(dest, hdr, tr); err != nil {
			return nil, err
		}
		extractedFiles = append(extractedFiles, dest)
	}
	return extractedFiles, nil
}

func extractFile(dest string, hdr *tar.Header, tr io.Reader) error {
	path := filepath.Join(dest, filepath.Clean(hdr.Name))
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	mode := hdr.FileInfo().Mode()
	uid := hdr.Uid
	gid := hdr.Gid

	whitelisted, err := CheckWhitelist(path)
	if err != nil {
		return err
	}
	if whitelisted && !checkWhitelistRoot(dest) {
		logrus.Debugf("Not adding %s because it is whitelisted", path)
		return nil
	}
	switch hdr.Typeflag {
	case tar.TypeReg:
		logrus.Debugf("creating file %s", path)
		// It's possible a file is in the tar before it's directory.
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logrus.Debugf("base %s for file %s does not exist. Creating.", base, path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		// Check if something already exists at path (symlinks etc.)
		// If so, delete it
		if FilepathExists(path) {
			if err := os.Remove(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new file.", path)
			}
		}
		currFile, err := os.Create(path)
		if err != nil {
			return err
		}
		// manually set permissions on file, since the default umask (022) will interfere
		if err = os.Chmod(path, mode); err != nil {
			return err
		}
		if _, err = io.Copy(currFile, tr); err != nil {
			return err
		}
		if err = currFile.Chown(uid, gid); err != nil {
			return err
		}
		currFile.Close()
	case tar.TypeDir:
		logrus.Debugf("creating dir %s", path)
		if err := os.MkdirAll(path, mode); err != nil {
			return err
		}
		// In some cases, MkdirAll doesn't change the permissions, so run Chmod
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
		if err := os.Chown(path, uid, gid); err != nil {
			return err
		}

	case tar.TypeLink:
		logrus.Debugf("link from %s to %s", hdr.Linkname, path)
		whitelisted, err := CheckWhitelist(hdr.Linkname)
		if err != nil {
			return err
		}
		if whitelisted {
			logrus.Debugf("skipping symlink from %s to %s because %s is whitelisted", hdr.Linkname, path, hdr.Linkname)
			return nil
		}
		// The base directory for a link may not exist before it is created.
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if FilepathExists(path) {
			if err := os.Remove(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new link", hdr.Name)
			}
		}

		if err := os.Link(filepath.Clean(filepath.Join("/", hdr.Linkname)), path); err != nil {
			return err
		}

	case tar.TypeSymlink:
		logrus.Debugf("symlink from %s to %s", hdr.Linkname, path)
		// The base directory for a symlink may not exist before it is created.
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if FilepathExists(path) {
			if err := os.Remove(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new symlink", hdr.Name)
			}
		}
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}
	}
	return nil
}

func CheckWhitelist(path string) (bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		logrus.Infof("unable to get absolute path for %s", path)
		return false, err
	}
	for _, wl := range whitelist {
		if HasFilepathPrefix(abs, wl.Path, wl.PrefixMatchOnly) {
			return true, nil
		}
	}
	return false, nil
}

func checkWhitelistRoot(root string) bool {
	if root == constants.RootDir {
		return false
	}
	for _, wl := range whitelist {
		if HasFilepathPrefix(root, wl.Path, wl.PrefixMatchOnly) {
			return true
		}
	}
	return false
}

// Get whitelist from roots of mounted files
// Each line of /proc/self/mountinfo is in the form:
// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
// (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)
// Where (5) is the mount point relative to the process's root
// From: https://www.kernel.org/doc/Documentation/filesystems/proc.txt
func fileSystemWhitelist(path string) ([]WhitelistEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		logrus.Debugf("Read the following line from %s: %s", path, line)
		if err != nil && err != io.EOF {
			return nil, err
		}
		lineArr := strings.Split(line, " ")
		if len(lineArr) < 5 {
			if err == io.EOF {
				logrus.Debugf("Reached end of file %s", path)
				break
			}
			continue
		}
		if lineArr[4] != constants.RootDir {
			logrus.Debugf("Appending %s from line: %s", lineArr[4], line)
			whitelist = append(whitelist, WhitelistEntry{
				Path:            lineArr[4],
				PrefixMatchOnly: false,
			})
		}
		if err == io.EOF {
			logrus.Debugf("Reached end of file %s", path)
			break
		}
	}
	return whitelist, nil
}

// RelativeFiles returns a list of all files at the filepath relative to root
func RelativeFiles(fp string, root string) ([]string, error) {
	var files []string
	fullPath := filepath.Join(root, fp)
	logrus.Debugf("Getting files and contents at root %s", fullPath)
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		whitelisted, err := CheckWhitelist(path)
		if err != nil {
			return err
		}
		if whitelisted && !HasFilepathPrefix(path, root, false) {
			return nil
		}
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	return files, err
}

// ParentDirectories returns a list of paths to all parent directories
// Ex. /some/temp/dir -> [/, /some, /some/temp, /some/temp/dir]
func ParentDirectories(path string) []string {
	path = filepath.Clean(path)
	dirs := strings.Split(path, "/")
	dirPath := constants.RootDir
	paths := []string{constants.RootDir}
	for index, dir := range dirs {
		if dir == "" || index == (len(dirs)-1) {
			continue
		}
		dirPath = filepath.Join(dirPath, dir)
		paths = append(paths, dirPath)
	}
	return paths
}

// FilepathExists returns true if the path exists
func FilepathExists(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

// CreateFile creates a file at path and copies over contents from the reader
func CreateFile(path string, reader io.Reader, perm os.FileMode, uid uint32, gid uint32) error {
	// Create directory path if it doesn't exist
	baseDir := filepath.Dir(path)
	if _, err := os.Lstat(baseDir); os.IsNotExist(err) {
		logrus.Debugf("baseDir %s for file %s does not exist. Creating.", baseDir, path)
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return err
		}
	}
	dest, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dest.Close()
	if _, err := io.Copy(dest, reader); err != nil {
		return err
	}
	if err := dest.Chmod(perm); err != nil {
		return err
	}
	return dest.Chown(int(uid), int(gid))
}

// AddVolumePathToWhitelist adds the given path to the whitelist with
// PrefixMatchOnly set to true. Snapshotting will ignore paths prefixed
// with the volume, but the volume itself will not be ignored.
func AddVolumePathToWhitelist(path string) error {
	logrus.Infof("adding volume %s to whitelist", path)
	whitelist = append(whitelist, WhitelistEntry{
		Path:            path,
		PrefixMatchOnly: true,
	})
	return nil
}

// DownloadFileToDest downloads the file at rawurl to the given dest for the ADD command
// From add command docs:
// 	1. If <src> is a remote file URL:
// 		- destination will have permissions of 0600
// 		- If remote file has HTTP Last-Modified header, we set the mtime of the file to that timestamp
func DownloadFileToDest(rawurl, dest string) error {
	resp, err := http.Get(rawurl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// TODO: set uid and gid according to current user
	if err := CreateFile(dest, resp.Body, 0600, 0, 0); err != nil {
		return err
	}
	mTime := time.Time{}
	lastMod := resp.Header.Get("Last-Modified")
	if lastMod != "" {
		if parsedMTime, err := http.ParseTime(lastMod); err == nil {
			mTime = parsedMTime
		}
	}
	return os.Chtimes(dest, mTime, mTime)
}

// CopyDir copies the file or directory at src to dest
// It returns a list of files it copied over
func CopyDir(src, dest string) ([]string, error) {
	files, err := RelativeFiles("", src)
	if err != nil {
		return nil, err
	}
	var copiedFiles []string
	for _, file := range files {
		fullPath := filepath.Join(src, file)
		fi, err := os.Lstat(fullPath)
		if err != nil {
			return nil, err
		}
		destPath := filepath.Join(dest, file)
		if fi.IsDir() {
			logrus.Debugf("Creating directory %s", destPath)

			uid := int(fi.Sys().(*syscall.Stat_t).Uid)
			gid := int(fi.Sys().(*syscall.Stat_t).Gid)

			if err := os.MkdirAll(destPath, fi.Mode()); err != nil {
				return nil, err
			}
			if err := os.Chown(destPath, uid, gid); err != nil {
				return nil, err
			}
		} else if fi.Mode()&os.ModeSymlink != 0 {
			// If file is a symlink, we want to create the same relative symlink
			if err := CopySymlink(fullPath, destPath); err != nil {
				return nil, err
			}
		} else {
			// ... Else, we want to copy over a file
			if err := CopyFile(fullPath, destPath); err != nil {
				return nil, err
			}
		}
		copiedFiles = append(copiedFiles, destPath)
	}
	return copiedFiles, nil
}

// CopySymlink copies the symlink at src to dest
func CopySymlink(src, dest string) error {
	link, err := os.Readlink(src)
	if err != nil {
		return err
	}
	linkDst := filepath.Join(dest, link)
	return os.Symlink(linkDst, dest)
}

// CopyFile copies the file at src to dest
func CopyFile(src, dest string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	logrus.Debugf("Copying file %s to %s", src, dest)
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	uid := fi.Sys().(*syscall.Stat_t).Uid
	gid := fi.Sys().(*syscall.Stat_t).Gid
	return CreateFile(dest, srcFile, fi.Mode(), uid, gid)
}

// HasFilepathPrefix checks if the given file path begins with prefix
func HasFilepathPrefix(path, prefix string, prefixMatchOnly bool) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	pathArray := strings.Split(path, "/")
	prefixArray := strings.Split(prefix, "/")

	if len(pathArray) < len(prefixArray) {
		return false
	}
	if prefixMatchOnly && len(pathArray) == len(prefixArray) {
		return false
	}
	for index := range prefixArray {
		if prefixArray[index] == pathArray[index] {
			continue
		}
		return false
	}
	return true
}
