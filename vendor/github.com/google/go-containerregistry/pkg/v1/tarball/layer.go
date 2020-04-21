// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tarball

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/go-containerregistry/pkg/v1/v1util"
)

type layer struct {
	digest      v1.Hash
	diffID      v1.Hash
	size        int64
	opener      Opener
	compressed  bool
	compression int
}

func (l *layer) Digest() (v1.Hash, error) {
	return l.digest, nil
}

func (l *layer) DiffID() (v1.Hash, error) {
	return l.diffID, nil
}

func (l *layer) Compressed() (io.ReadCloser, error) {
	rc, err := l.opener()
	if err == nil && !l.compressed {
		return v1util.GzipReadCloserLevel(rc, l.compression), nil
	}

	return rc, err
}

func (l *layer) Uncompressed() (io.ReadCloser, error) {
	rc, err := l.opener()
	if err == nil && l.compressed {
		return v1util.GunzipReadCloser(rc)
	}

	return rc, err
}

func (l *layer) Size() (int64, error) {
	return l.size, nil
}

func (l *layer) MediaType() (types.MediaType, error) {
	return types.DockerLayer, nil
}

// LayerOption applies options to layer
type LayerOption func(*layer)

// WithCompressionLevel sets the gzip compression. See `gzip.NewWriterLevel` for possible values.
func WithCompressionLevel(level int) LayerOption {
	return func(l *layer) {
		l.compression = level
	}
}

// LayerFromFile returns a v1.Layer given a tarball
func LayerFromFile(path string, opts ...LayerOption) (v1.Layer, error) {
	opener := func() (io.ReadCloser, error) {
		return os.Open(path)
	}
	return LayerFromOpener(opener, opts...)
}

// LayerFromOpener returns a v1.Layer given an Opener function
func LayerFromOpener(opener Opener, opts ...LayerOption) (v1.Layer, error) {
	rc, err := opener()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	compressed, err := v1util.IsGzipped(rc)
	if err != nil {
		return nil, err
	}

	layer := &layer{
		compressed:  compressed,
		compression: gzip.BestSpeed,
		opener:      opener,
	}

	for _, opt := range opts {
		opt(layer)
	}

	if layer.digest, layer.size, err = computeDigest(opener, compressed, layer.compression); err != nil {
		return nil, err
	}

	if layer.diffID, err = computeDiffID(opener, compressed); err != nil {
		return nil, err
	}

	return layer, nil
}

// LayerFromReader returns a v1.Layer given a io.Reader.
func LayerFromReader(reader io.Reader, opts ...LayerOption) (v1.Layer, error) {
	// Buffering due to Opener requiring multiple calls.
	a, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return LayerFromOpener(func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(a)), nil
	}, opts...)
}

func computeDigest(opener Opener, compressed bool, compression int) (v1.Hash, int64, error) {
	rc, err := opener()
	if err != nil {
		return v1.Hash{}, 0, err
	}
	defer rc.Close()

	if compressed {
		return v1.SHA256(rc)
	}

	return v1.SHA256(v1util.GzipReadCloserLevel(ioutil.NopCloser(rc), compression))
}

func computeDiffID(opener Opener, compressed bool) (v1.Hash, error) {
	rc, err := opener()
	if err != nil {
		return v1.Hash{}, err
	}
	defer rc.Close()

	if !compressed {
		digest, _, err := v1.SHA256(rc)
		return digest, err
	}

	reader, err := gzip.NewReader(rc)
	if err != nil {
		return v1.Hash{}, err
	}

	diffID, _, err := v1.SHA256(reader)
	return diffID, err
}
