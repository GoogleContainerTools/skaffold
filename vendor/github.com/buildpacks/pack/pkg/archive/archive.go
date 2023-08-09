// Package archive defines a set of functions for reading and writing directories and files in a number of tar formats.
package archive // import "github.com/buildpacks/pack/pkg/archive"

import (
	"archive/tar"
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"
)

var NormalizedDateTime time.Time

func init() {
	NormalizedDateTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)
}

type TarWriter interface {
	WriteHeader(hdr *tar.Header) error
	Write(b []byte) (int, error)
	Close() error
}

type TarWriterFactory interface {
	NewWriter(io.Writer) TarWriter
}

type defaultTarWriterFactory struct{}

func DefaultTarWriterFactory() TarWriterFactory {
	return defaultTarWriterFactory{}
}

func (defaultTarWriterFactory) NewWriter(w io.Writer) TarWriter {
	return tar.NewWriter(w)
}

func ReadDirAsTar(srcDir, basePath string, uid, gid int, mode int64, normalizeModTime, includeRoot bool, fileFilter func(string) bool) io.ReadCloser {
	return GenerateTar(func(tw TarWriter) error {
		return WriteDirToTar(tw, srcDir, basePath, uid, gid, mode, normalizeModTime, includeRoot, fileFilter)
	})
}

func ReadZipAsTar(srcPath, basePath string, uid, gid int, mode int64, normalizeModTime bool, fileFilter func(string) bool) io.ReadCloser {
	return GenerateTar(func(tw TarWriter) error {
		return WriteZipToTar(tw, srcPath, basePath, uid, gid, mode, normalizeModTime, fileFilter)
	})
}

func GenerateTar(genFn func(TarWriter) error) io.ReadCloser {
	return GenerateTarWithWriter(genFn, DefaultTarWriterFactory())
}

// GenerateTarWithTar returns a reader to a tar from a generator function using a writer from the provided factory.
// Note that the generator will not fully execute until the reader is fully read from. Any errors returned by the
// generator will be returned when reading the reader.
func GenerateTarWithWriter(genFn func(TarWriter) error, twf TarWriterFactory) io.ReadCloser {
	errChan := make(chan error)
	pr, pw := io.Pipe()

	go func() {
		tw := twf.NewWriter(pw)
		defer func() {
			if r := recover(); r != nil {
				tw.Close()
				pw.CloseWithError(errors.Errorf("panic: %v", r))
			}
		}()

		err := genFn(tw)

		closeErr := tw.Close()
		closeErr = aggregateError(closeErr, pw.CloseWithError(err))

		errChan <- closeErr
	}()

	closed := false
	return ioutils.NewReadCloserWrapper(pr, func() error {
		if closed {
			return errors.New("reader already closed")
		}

		var completeErr error

		// closing the reader ensures that if anything attempts
		// further reading it doesn't block waiting for content
		if err := pr.Close(); err != nil {
			completeErr = aggregateError(completeErr, err)
		}

		// wait until everything closes properly
		if err := <-errChan; err != nil {
			completeErr = aggregateError(completeErr, err)
		}

		closed = true
		return completeErr
	})
}

func aggregateError(base, addition error) error {
	if addition == nil {
		return base
	}

	if base == nil {
		return addition
	}

	return errors.Wrap(addition, base.Error())
}

func CreateSingleFileTarReader(path, txt string) io.ReadCloser {
	tarBuilder := TarBuilder{}
	tarBuilder.AddFile(path, 0644, NormalizedDateTime, []byte(txt))
	return tarBuilder.Reader(DefaultTarWriterFactory())
}

func CreateSingleFileTar(tarFile, path, txt string) error {
	tarBuilder := TarBuilder{}
	tarBuilder.AddFile(path, 0644, NormalizedDateTime, []byte(txt))
	return tarBuilder.WriteToPath(tarFile, DefaultTarWriterFactory())
}

// ErrEntryNotExist is an error returned if an entry path doesn't exist
var ErrEntryNotExist = errors.New("not exist")

// IsEntryNotExist detects whether a given error is of type ErrEntryNotExist
func IsEntryNotExist(err error) bool {
	return err == ErrEntryNotExist || errors.Cause(err) == ErrEntryNotExist
}

// ReadTarEntry reads and returns a tar file
func ReadTarEntry(rc io.Reader, entryPath string) (*tar.Header, []byte, error) {
	tr := tar.NewReader(rc)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get next tar entry")
		}

		if path.Clean(header.Name) == entryPath {
			buf, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to read contents of '%s'", entryPath)
			}

			return header, buf, nil
		}
	}

	return nil, nil, errors.Wrapf(ErrEntryNotExist, "could not find entry path '%s'", entryPath)
}

// WriteDirToTar writes the contents of a directory to a tar writer. `basePath` is the "location" in the tar the
// contents will be placed. The includeRoot param sets the permissions and metadata on the root file.
func WriteDirToTar(tw TarWriter, srcDir, basePath string, uid, gid int, mode int64, normalizeModTime, includeRoot bool, fileFilter func(string) bool) error {
	if includeRoot {
		rootHeader := &tar.Header{
			Typeflag: tar.TypeDir,
			Name:     basePath,
			Mode:     mode,
		}
		finalizeHeader(rootHeader, uid, gid, mode, normalizeModTime)
		if err := tw.WriteHeader(rootHeader); err != nil {
			return err
		}
	}

	return filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		var relPath string
		if fileFilter != nil {
			relPath, err = filepath.Rel(srcDir, file)
			if err != nil {
				return err
			}
			if !fileFilter(relPath) {
				return nil
			}
		}
		if err != nil {
			return err
		}

		if fi.Mode()&os.ModeSocket != 0 {
			return nil
		}

		var header *tar.Header
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(file)
			if err != nil {
				return err
			}

			// Ensure that symlinks have Linux link names, independent of source OS
			header, err = tar.FileInfoHeader(fi, filepath.ToSlash(target))
			if err != nil {
				return err
			}
		} else {
			header, err = tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}
		}

		if relPath == "" {
			relPath, err = filepath.Rel(srcDir, file)
			if err != nil {
				return err
			}
		}
		if relPath == "." {
			return nil
		}

		header.Name = filepath.ToSlash(filepath.Join(basePath, relPath))
		finalizeHeader(header, uid, gid, mode, normalizeModTime)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.Mode().IsRegular() {
			f, err := os.Open(filepath.Clean(file))
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

// WriteZipToTar writes the contents of a zip file to a tar writer.
func WriteZipToTar(tw TarWriter, srcZip, basePath string, uid, gid int, mode int64, normalizeModTime bool, fileFilter func(string) bool) error {
	zipReader, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	var fileMode int64
	for _, f := range zipReader.File {
		if fileFilter != nil && !fileFilter(f.Name) {
			continue
		}

		fileMode = mode
		if isFatFile(f.FileHeader) {
			fileMode = 0777
		}

		var header *tar.Header
		if f.Mode()&os.ModeSymlink != 0 {
			target, err := func() (string, error) {
				r, err := f.Open()
				if err != nil {
					return "", nil
				}
				defer r.Close()

				// contents is the target of the symlink
				target, err := ioutil.ReadAll(r)
				if err != nil {
					return "", err
				}

				return string(target), nil
			}()

			if err != nil {
				return err
			}

			header, err = tar.FileInfoHeader(f.FileInfo(), target)
			if err != nil {
				return err
			}
		} else {
			header, err = tar.FileInfoHeader(f.FileInfo(), f.Name)
			if err != nil {
				return err
			}
		}

		header.Name = filepath.ToSlash(filepath.Join(basePath, f.Name))
		finalizeHeader(header, uid, gid, fileMode, normalizeModTime)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if f.Mode().IsRegular() {
			err := func() error {
				fi, err := f.Open()
				if err != nil {
					return err
				}
				defer fi.Close()

				_, err = io.Copy(tw, fi)
				return err
			}()

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func isFatFile(header zip.FileHeader) bool {
	var (
		creatorFAT  uint16 = 0
		creatorVFAT uint16 = 14
	)

	// This identifies FAT files, based on the `zip` source: https://golang.org/src/archive/zip/struct.go
	firstByte := header.CreatorVersion >> 8
	return firstByte == creatorFAT || firstByte == creatorVFAT
}

func finalizeHeader(header *tar.Header, uid, gid int, mode int64, normalizeModTime bool) {
	NormalizeHeader(header, normalizeModTime)
	if mode != -1 {
		header.Mode = mode
	}
	header.Uid = uid
	header.Gid = gid
}

// NormalizeHeader normalizes a tar.Header
//
// Normalizes the following:
// 	- ModTime
// 	- GID
// 	- UID
// 	- User Name
// 	- Group Name
func NormalizeHeader(header *tar.Header, normalizeModTime bool) {
	if normalizeModTime {
		header.ModTime = NormalizedDateTime
	}
	header.Uid = 0
	header.Gid = 0
	header.Uname = ""
	header.Gname = ""
}

// IsZip detects whether or not a File is a zip directory
func IsZip(path string) (bool, error) {
	r, err := zip.OpenReader(path)

	switch {
	case err == nil:
		r.Close()
		return true, nil
	case err == zip.ErrFormat:
		return false, nil
	default:
		return false, err
	}
}
