package archive

import (
	"archive/tar"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

type TarBuilder struct {
	files []fileEntry
}

type fileEntry struct {
	typeFlag byte
	path     string
	mode     int64
	modTime  time.Time
	contents []byte
}

func (t *TarBuilder) AddFile(path string, mode int64, modTime time.Time, contents []byte) {
	t.files = append(t.files, fileEntry{
		typeFlag: tar.TypeReg,
		path:     path,
		mode:     mode,
		modTime:  modTime,
		contents: contents,
	})
}

func (t *TarBuilder) AddDir(path string, mode int64, modTime time.Time) {
	t.files = append(t.files, fileEntry{
		typeFlag: tar.TypeDir,
		path:     path,
		mode:     mode,
		modTime:  modTime,
	})
}

func (t *TarBuilder) Reader() io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		var err error
		defer func() {
			pw.CloseWithError(err)
		}()
		_, err = t.WriteTo(pw)
	}()

	return pr
}

func (t *TarBuilder) WriteToPath(path string) error {
	fh, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "create file for tar: %s", style.Symbol(path))
	}
	defer fh.Close()

	_, err = t.WriteTo(fh)
	return err
}

func (t *TarBuilder) WriteTo(writer io.Writer) (int64, error) {
	tw := tar.NewWriter(writer)
	defer tw.Close()

	var written int64
	for _, f := range t.files {
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: f.typeFlag,
			Name:     f.path,
			Size:     int64(len(f.contents)),
			Mode:     f.mode,
			ModTime:  f.modTime,
		}); err != nil {
			return written, err
		}

		n, err := tw.Write(f.contents)
		if err != nil {
			return written, err
		}

		written += int64(n)
	}

	return written, nil
}
