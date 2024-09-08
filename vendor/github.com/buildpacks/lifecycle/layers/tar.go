package layers

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"

	"github.com/buildpacks/lifecycle/archive"
)

// TarLayer creates a layer from the provided tar path.
// The provided tar may be compressed or uncompressed.
// TarLayer will return a layer with uncompressed data and zeroed out timestamps.
// FIXME: investigate if it's possible to set "no compression" in kaniko, then we can avoid decompressing layers
func (f *Factory) TarLayer(withID string, fromTarPath string, createdBy string) (layer Layer, err error) {
	var tarReader archive.TarReader
	layerReader, err := os.Open(fromTarPath) // #nosec G304
	if err != nil {
		return Layer{}, err
	}
	defer layerReader.Close() // nolint

	tarReader, err = tryCompressedReader(layerReader)
	if err != nil {
		// uncompressed reader
		layerReader, err = os.Open(fromTarPath) // #nosec G304
		if err != nil {
			return Layer{}, err
		}
		defer layerReader.Close() // nolint
		tarReader = tar.NewReader(layerReader)
	}
	return f.writeLayer(withID, createdBy, func(tw *archive.NormalizingTarWriter) error {
		return copyTar(tw, tarReader)
	})
}

func tryCompressedReader(fromReader io.ReadCloser) (archive.TarReader, error) {
	zr, err := gzip.NewReader(fromReader)
	if err != nil {
		_ = fromReader.Close()
		return nil, err
	}
	return tar.NewReader(zr), nil
}

func copyTar(dst archive.TarWriter, src archive.TarReader) error {
	var (
		header *tar.Header
		err    error
	)
	for {
		if header, err = src.Next(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err = dst.WriteHeader(header); err != nil {
			return err
		}
		if _, err = io.Copy(dst, src); err != nil {
			return err
		}
	}
}
