package layers

import (
	"archive/tar"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/buildpacks/lifecycle/archive"
)

type layerWriter struct {
	io.Writer
	io.Closer
	hasher *concurrentHasher
	path   string
}

func newFileLayerWriter(dest string) (*layerWriter, error) {
	hasher := newConcurrentHasher(sha256.New())
	file, err := os.Create(dest)
	if err != nil {
		return nil, err
	}
	w := io.MultiWriter(hasher, file)
	return &layerWriter{w, file, hasher, dest}, nil
}

func (lw *layerWriter) Digest() string {
	return fmt.Sprintf("sha256:%x", lw.hasher.Sum(nil))
}

func tarWriter(lw *layerWriter) *archive.NormalizingTarWriter {
	tw := archive.NewNormalizingTarWriter(tar.NewWriter(lw))
	tw.WithModTime(archive.NormalizedModTime)
	return tw
}
