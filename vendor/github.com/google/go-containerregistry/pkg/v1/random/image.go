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

package random

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// uncompressedLayer implements partial.UncompressedLayer from raw bytes.
// TODO(mattmoor): Consider moving this into a library.
type uncompressedLayer struct {
	diffID  v1.Hash
	content []byte
}

// DiffID implements partial.UncompressedLayer
func (ul *uncompressedLayer) DiffID() (v1.Hash, error) {
	return ul.diffID, nil
}

// Uncompressed implements partial.UncompressedLayer
func (ul *uncompressedLayer) Uncompressed() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewBuffer(ul.content)), nil
}

var _ partial.UncompressedLayer = (*uncompressedLayer)(nil)

// Image returns a pseudo-randomly generated Image.
func Image(byteSize, layers int64) (v1.Image, error) {
	layerz := make(map[v1.Hash]partial.UncompressedLayer)
	for i := int64(0); i < layers; i++ {
		var b bytes.Buffer
		tw := tar.NewWriter(&b)
		if err := tw.WriteHeader(&tar.Header{
			Name:     fmt.Sprintf("random_file_%d.txt", i),
			Size:     byteSize,
			Typeflag: tar.TypeRegA,
		}); err != nil {
			return nil, err
		}
		if _, err := io.CopyN(tw, rand.Reader, byteSize); err != nil {
			return nil, err
		}
		if err := tw.Close(); err != nil {
			return nil, err
		}
		bts := b.Bytes()
		h, _, err := v1.SHA256(bytes.NewReader(bts))
		if err != nil {
			return nil, err
		}
		layerz[h] = &uncompressedLayer{
			diffID:  h,
			content: bts,
		}
	}

	cfg := &v1.ConfigFile{}

	// Some clients check this.
	cfg.RootFS.Type = "layers"

	// It is ok that iteration order is random in Go, because this is the random image anyways.
	for k := range layerz {
		cfg.RootFS.DiffIDs = append(cfg.RootFS.DiffIDs, k)
	}

	for i := int64(0); i < layers; i++ {
		cfg.History = append(cfg.History, v1.History{
			Author:    "random.Image",
			Comment:   fmt.Sprintf("this is a random history %d", i),
			CreatedBy: "random",
			Created:   v1.Time{time.Now()},
		})
	}

	return partial.UncompressedToImage(&image{
		config: cfg,
		layers: layerz,
	})
}

// image is pseudo-randomly generated.
type image struct {
	config *v1.ConfigFile
	layers map[v1.Hash]partial.UncompressedLayer
}

var _ partial.UncompressedImageCore = (*image)(nil)

// RawConfigFile implements partial.UncompressedImageCore
func (i *image) RawConfigFile() ([]byte, error) {
	return partial.RawConfigFile(i)
}

// ConfigFile implements v1.Image
func (i *image) ConfigFile() (*v1.ConfigFile, error) {
	return i.config, nil
}

// MediaType implements partial.UncompressedImageCore
func (i *image) MediaType() (types.MediaType, error) {
	return types.DockerManifestSchema2, nil
}

// LayerByDiffID implements partial.UncompressedImageCore
func (i *image) LayerByDiffID(diffID v1.Hash) (partial.UncompressedLayer, error) {
	l, ok := i.layers[diffID]
	if !ok {
		return nil, fmt.Errorf("unknown diff_id: %v", diffID)
	}
	return l, nil
}
