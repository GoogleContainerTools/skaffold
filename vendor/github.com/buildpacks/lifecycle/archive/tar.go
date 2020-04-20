package archive

import (
	"archive/tar"
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func WriteFilesToTar(dest string, uid, gid int, files ...string) (string, map[string]struct{}, error) {
	hasher := newConcurrentHasher(sha256.New())
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

	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), fileSet, nil
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
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := newConcurrentHasher(sha256.New())
	w := bufio.NewWriterSize(io.MultiWriter(hasher, f), 1024*1024)

	if err := WriteTarArchive(w, sourceDir, uid, gid); err != nil {
		return "", err
	}

	if err := w.Flush(); err != nil {
		return "", err
	}

	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), nil
}

func WriteTarArchive(w io.Writer, srcDir string, uid, gid int) error {
	srcDir = filepath.Clean(srcDir)

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
	// Avoid umask from changing the file permissions in the tar file.
	umask := setUmask(0)
	defer setUmask(umask)

	buf := make([]byte, 32*32*1024)
	dirsFound := make(map[string]bool)

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
			dirsFound[path] = true

		case tar.TypeReg, tar.TypeRegA:
			dirPath := filepath.Dir(path)
			if !dirsFound[dirPath] {
				if _, err := os.Stat(dirPath); os.IsNotExist(err) {
					if err := os.MkdirAll(dirPath, applyUmask(os.ModePerm, umask)); err != nil {
						return err
					}
					dirsFound[dirPath] = true
				}
			}

			if err := writeFile(tr, path, hdr.FileInfo().Mode(), buf); err != nil {
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

func applyUmask(mode os.FileMode, umask int) os.FileMode {
	return os.FileMode(int(mode) &^ umask)
}

func writeFile(in io.Reader, path string, mode os.FileMode, buf []byte) error {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()
	_, err = io.CopyBuffer(fh, in, buf)
	return err
}
