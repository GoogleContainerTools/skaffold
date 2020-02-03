package archive

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func WriteFilesToTar(dest string, uid, gid int, files ...string) (string, map[string]struct{}, error) {
	hasher := sha256.New()
	f, err := os.Create(dest)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	w := io.MultiWriter(hasher, f)
	tw := tar.NewWriter(w)

	fileSet := map[string]struct{}{}
	for _, file := range files {
		if AddFileToArchive(tw, file, uid, gid, fileSet) != nil {
			return "", nil, err
		}
	}
	_ = tw.Close()

	sha := hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size())))
	return "sha256:" + sha, fileSet, nil
}

func AddFileToArchive(tw *tar.Writer, srcDir string, uid, gid int, fileSet map[string]struct{}) error {
	err := addParentDirsUnique(srcDir, tw, uid, gid, fileSet)
	if err != nil {
		return err
	}

	return filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if _, ok := fileSet[file]; ok {
			return nil
		}
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSocket != 0 {
			return nil
		}
		var header *tar.Header
		var target string
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err = os.Readlink(file)
			if err != nil {
				return err
			}
		}
		header, err = tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
		header.Name = file
		header.ModTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)
		header.Uid = uid
		header.Gid = gid
		header.Uname = ""
		header.Gname = ""

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}

		fileSet[file] = struct{}{}
		return nil
	})
}

func WriteTarFile(sourceDir, dest string, uid, gid int) (string, error) {
	hasher := sha256.New()
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()
	w := io.MultiWriter(hasher, f)

	if err := WriteTarArchive(w, sourceDir, uid, gid); err != nil {
		return "", err
	}
	sha := hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size())))
	return "sha256:" + sha, nil
}

func WriteTarArchive(w io.Writer, srcDir string, uid, gid int) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	err := addParentDirs(srcDir, tw, uid, gid)
	if err != nil {
		return err
	}

	return filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSocket != 0 {
			return nil
		}
		var header *tar.Header
		var target string
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err = os.Readlink(file)
			if err != nil {
				return err
			}
		}
		header, err = tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
		header.Name = file
		header.ModTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)
		header.Uid = uid
		header.Gid = gid
		header.Uname = ""
		header.Gname = ""

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})
}

func addParentDirsUnique(tarDir string, tw *tar.Writer, uid, gid int, parentDirs map[string]struct{}) error {
	parent := filepath.Dir(tarDir)
	if parent == "." || parent == "/" {
		return nil
	}

	if _, ok := parentDirs[parent]; ok {
		return nil
	}

	if err := addParentDirsUnique(parent, tw, uid, gid, parentDirs); err != nil {
		return err
	}

	info, err := os.Stat(parent)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, parent)
	if err != nil {
		return err
	}
	header.Name = parent
	header.ModTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)

	parentDirs[parent] = struct{}{}

	return tw.WriteHeader(header)
}

func addParentDirs(tarDir string, tw *tar.Writer, uid, gid int) error {
	parent := filepath.Dir(tarDir)
	if parent == "." || parent == "/" {
		return nil
	}

	if err := addParentDirs(parent, tw, uid, gid); err != nil {
		return err
	}

	info, err := os.Stat(parent)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, parent)
	if err != nil {
		return err
	}
	header.Name = parent
	header.ModTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)

	return tw.WriteHeader(header)
}

type PathMode struct {
	Path string
	Mode os.FileMode
}

func Untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	var pathModes []PathMode
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			for _, pathMode := range pathModes {
				if err := os.Chmod(pathMode.Path, pathMode.Mode); err != nil {
					return err
				}
			}
			return nil
		}
		if err != nil {
			return err
		}

		path := filepath.Join(dest, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(path); os.IsNotExist(err) {
				pathMode := PathMode{path, hdr.FileInfo().Mode()}
				pathModes = append(pathModes, pathMode)
			}
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			_, err := os.Stat(filepath.Dir(path))
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
					return err
				}
			}
			if err := writeFile(tr, path, hdr.FileInfo().Mode()); err != nil {
				return err
			}
			// Update permissions in case umask was applied.
			if err := os.Chmod(path, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown file type in tar %d", hdr.Typeflag)
		}
	}
}

func writeFile(in io.Reader, path string, mode os.FileMode) error {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()
	_, err = io.Copy(fh, in)
	return err
}
