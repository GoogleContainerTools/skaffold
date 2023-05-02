package layer

import (
	"archive/tar"
	"fmt"
	"io"

	ilayer "github.com/buildpacks/imgutil/layer"

	"github.com/buildpacks/pack/pkg/archive"
)

type WriterFactory struct {
	os string
}

func NewWriterFactory(imageOS string) (*WriterFactory, error) {
	if imageOS != "linux" && imageOS != "windows" {
		return nil, fmt.Errorf("provided image OS '%s' must be either 'linux' or 'windows'", imageOS)
	}

	return &WriterFactory{os: imageOS}, nil
}

func (f *WriterFactory) NewWriter(fileWriter io.Writer) archive.TarWriter {
	if f.os == "windows" {
		return ilayer.NewWindowsWriter(fileWriter)
	}

	// Linux images use tar.Writer
	return tar.NewWriter(fileWriter)
}
