package layer

import (
	"archive/tar"
	"io"

	"github.com/buildpacks/imgutil"
	ilayer "github.com/buildpacks/imgutil/layer"

	"github.com/buildpacks/pack/internal/archive"
)

type WriterFactory struct {
	os string
}

func NewWriterFactory(image imgutil.Image) (*WriterFactory, error) {
	os, err := image.OS()
	if err != nil {
		return nil, err
	}

	return &WriterFactory{os: os}, nil
}

func (f *WriterFactory) NewWriter(fileWriter io.Writer) archive.TarWriter {
	if f.os == "windows" {
		return ilayer.NewWindowsWriter(fileWriter)
	}

	// Linux images use tar.Writer
	return tar.NewWriter(fileWriter)
}
