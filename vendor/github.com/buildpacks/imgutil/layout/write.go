package layout

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"golang.org/x/sync/errgroup"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
)

type AppendOption func(*appendOptions)

type appendOptions struct {
	withoutLayers bool
	annotations   map[string]string
}

func WithoutLayers() AppendOption {
	return func(i *appendOptions) {
		i.withoutLayers = true
	}
}

func WithAnnotations(annotations map[string]string) AppendOption {
	return func(i *appendOptions) {
		i.annotations = annotations
	}
}

// AppendImage mimics GGCR's AppendImage in that it appends an image to a `layout.Path`,
// but the image appended does not include any layers in the `blobs` directory.
// The returned image will return layers when Layers(), LayerByDiffID(), or LayerByDigest() are called,
// but the returned layer will error when DiffID(), Compressed(), or Uncompressed() are called.
// This is useful when we need to satisfy the v1.Image interface but do not need to access any layers.
func (l Path) AppendImage(img v1.Image, ops ...AppendOption) error {
	o := &appendOptions{}
	for _, op := range ops {
		op(o)
	}
	annotations := map[string]string{}
	if o.annotations != nil {
		annotations = o.annotations
	}

	if o.withoutLayers {
		return l.writeImageWithoutLayers(img, annotations)
	}
	return l.appendImage(img, annotations)
}

// writeImageWithoutLayers is the same implementation of ggcr layout writeImage method, removing the writeLayer code
func (l Path) writeImageWithoutLayers(img v1.Image, annotations map[string]string) error {
	if err := l.writeImage(img); err != nil {
		return err
	}

	mt, err := img.MediaType()
	if err != nil {
		return err
	}

	d, err := img.Digest()
	if err != nil {
		return err
	}

	manifest, err := img.RawManifest()
	if err != nil {
		return err
	}

	desc := v1.Descriptor{
		MediaType:   mt,
		Size:        int64(len(manifest)),
		Digest:      d,
		Annotations: annotations,
	}
	return l.AppendDescriptor(desc)
}

func (l Path) appendImage(img v1.Image, annotations map[string]string) error {
	layers, err := img.Layers()
	if err != nil {
		return err
	}

	// Write the layers concurrently.
	var g errgroup.Group
	for _, layer := range layers {
		layer := layer
		g.Go(func() error {
			return l.writeLayer(layer)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return l.writeImageWithoutLayers(img, annotations)
}

func (l Path) writeImage(img v1.Image) error {
	// Write the config.
	cfgName, err := img.ConfigName()
	if err != nil {
		return err
	}
	cfgBlob, err := img.RawConfigFile()
	if err != nil {
		return err
	}
	if err := l.WriteBlob(cfgName, io.NopCloser(bytes.NewReader(cfgBlob))); err != nil {
		return err
	}

	// Write the img manifest.
	d, err := img.Digest()
	if err != nil {
		return err
	}
	manifest, err := img.RawManifest()
	if err != nil {
		return err
	}
	return l.WriteBlob(d, io.NopCloser(bytes.NewReader(manifest)))
}

func Write(path string, ii v1.ImageIndex) (Path, error) {
	layoutPath, err := layout.Write(path, ii)
	if err != nil {
		return Path{}, err
	}

	return Path{Path: layoutPath}, nil
}

func FromPath(path string) (Path, error) {
	layoutPath, err := layout.FromPath(path)

	if err != nil {
		return Path{}, err
	}

	return Path{Path: layoutPath}, nil
}

// writeLayer is the same internal implementation from ggcr layout package, but because it is calling an internal
// writeBlob method we need to override we copied here.
func (l Path) writeLayer(layer v1.Layer) error {
	d, err := layer.Digest()

	if errors.Is(err, stream.ErrNotComputed) {
		// Allow digest errors, since streams may not have calculated the hash
		// yet. Instead, use an empty value, which will be transformed into a
		// random file name with `ioutil.TempFile` and the final digest will be
		// calculated after writing to a temp file and before renaming to the
		// final path.
		d = v1.Hash{Algorithm: "sha256", Hex: ""}
	} else if err != nil {
		return err
	}

	s, err := layer.Size()
	if errors.Is(err, stream.ErrNotComputed) {
		// Allow size errors, since streams may not have calculated the size
		// yet. Instead, use zero as a sentinel value meaning that no size
		// comparison can be done and any sized blob file should be considered
		// valid and not overwritten.
		//
		// TODO: Provide an option to always overwrite blobs.
		s = -1
	} else if err != nil {
		return err
	}

	r, err := layer.Compressed()
	if err != nil {
		return err
	}

	if err := l.writeBlob(d, s, r, layer.Digest); err != nil {
		return fmt.Errorf("error writing layer: %w", err)
	}
	return nil
}

// writeBlob ggcr implementation was modified to skip the blob when it returns a size of zero.
// See layout.Image.Layers() method
func (l Path) writeBlob(hash v1.Hash, size int64, rc io.ReadCloser, renamer func() (v1.Hash, error)) error {
	if hash.Hex == "" && renamer == nil {
		panic("writeBlob called an invalid hash and no renamer")
	}

	dir := l.append("blobs", hash.Algorithm)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	// Check if blob already exists and is the correct size
	file := filepath.Join(dir, hash.Hex)
	if s, err := os.Stat(file); err == nil && !s.IsDir() && (s.Size() == size || size == -1) {
		return nil
	}

	// If a renamer func was provided write to a temporary file
	open := func() (*os.File, error) { return os.Create(file) }
	if renamer != nil {
		open = func() (*os.File, error) { return ioutil.TempFile(dir, hash.Hex) }
	}
	w, err := open()
	if err != nil {
		return err
	}
	if renamer != nil {
		// Delete temp file if an error is encountered before renaming
		defer func() {
			if err := os.Remove(w.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
				logs.Warn.Printf("error removing temporary file after encountering an error while writing blob: %v", err)
			}
		}()
	}
	defer w.Close()

	// Write to file and exit if not renaming
	var skip = false
	if n, err := io.Copy(w, rc); err != nil || renamer == nil {
		return err
	} else if size != -1 && n != size {
		if n != 0 {
			return fmt.Errorf("expected blob size %d, but only wrote %d", size, n)
		}
		// When the blob size was 0 we want to skip it
		skip = true
	}

	// Always close reader before renaming, since Close computes the digest in
	// the case of streaming layers. If Close is not called explicitly, it will
	// occur in a goroutine that is not guaranteed to succeed before renamer is
	// called. When renamer is the layer's Digest method, it can return
	// ErrNotComputed.
	if err := rc.Close(); err != nil {
		return err
	}

	// Always close file before renaming
	if err := w.Close(); err != nil {
		return err
	}

	// Remove the empty blob when is skipped
	if skip {
		os.Remove(file)
		return nil
	}

	// Rename file based on the final hash
	finalHash, err := renamer()
	if err != nil {
		return fmt.Errorf("error getting final digest of layer: %w", err)
	}

	renamePath := l.append("blobs", finalHash.Algorithm, finalHash.Hex)
	return os.Rename(w.Name(), renamePath)
}
