package blob

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/archive"
)

type Blob interface {
	Open() (io.ReadCloser, error)
}

type blob struct {
	path string
}

func NewBlob(path string) Blob {
	return &blob{path: path}
}

// Open returns an io.ReadCloser whose contents are in tar archive format
func (b blob) Open() (r io.ReadCloser, err error) {
	fi, err := os.Stat(b.path)
	if err != nil {
		return nil, errors.Wrapf(err, "read blob at path '%s'", b.path)
	}
	if fi.IsDir() {
		return archive.ReadDirAsTar(b.path, ".", 0, 0, -1, true), nil
	}

	fh, err := os.Open(b.path)
	if err != nil {
		return nil, errors.Wrap(err, "open buildpack archive")
	}
	defer func() {
		if err != nil {
			fh.Close()
		}
	}()

	if ok, err := isGZip(fh); err != nil {
		return nil, errors.Wrap(err, "check header")
	} else if !ok {
		return fh, nil
	}
	gzr, err := gzip.NewReader(fh)
	if err != nil {
		return nil, errors.Wrap(err, "create gzip reader")
	}

	rc := ioutils.NewReadCloserWrapper(gzr, func() error {
		defer fh.Close()
		return gzr.Close()
	})

	return rc, nil
}

func isGZip(file io.ReadSeeker) (bool, error) {
	b := make([]byte, 3)
	if _, err := file.Seek(0, 0); err != nil {
		return false, err
	}
	_, err := file.Read(b)
	if err != nil && err != io.EOF {
		return false, err
	} else if err == io.EOF {
		return false, nil
	}
	if _, err := file.Seek(0, 0); err != nil {
		return false, err
	}
	return bytes.Equal(b, []byte("\x1f\x8b\x08")), nil
}
