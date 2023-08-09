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
	"bytes"
	"encoding/json"
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type randomIndex struct {
	images   map[v1.Hash]v1.Image
	manifest *v1.IndexManifest
}

// Index returns a pseudo-randomly generated ImageIndex with count images, each
// having the given number of layers of size byteSize.
func Index(byteSize, layers, count int64) (v1.ImageIndex, error) {
	manifest := v1.IndexManifest{
		SchemaVersion: 2,
		MediaType:     types.OCIImageIndex,
		Manifests:     []v1.Descriptor{},
	}

	images := make(map[v1.Hash]v1.Image)
	for i := int64(0); i < count; i++ {
		img, err := Image(byteSize, layers)
		if err != nil {
			return nil, err
		}

		rawManifest, err := img.RawManifest()
		if err != nil {
			return nil, err
		}
		digest, size, err := v1.SHA256(bytes.NewReader(rawManifest))
		if err != nil {
			return nil, err
		}
		mediaType, err := img.MediaType()
		if err != nil {
			return nil, err
		}

		manifest.Manifests = append(manifest.Manifests, v1.Descriptor{
			Digest:    digest,
			Size:      size,
			MediaType: mediaType,
		})

		images[digest] = img
	}

	return &randomIndex{
		images:   images,
		manifest: &manifest,
	}, nil
}

func (i *randomIndex) MediaType() (types.MediaType, error) {
	return i.manifest.MediaType, nil
}

func (i *randomIndex) Digest() (v1.Hash, error) {
	return partial.Digest(i)
}

func (i *randomIndex) Size() (int64, error) {
	return partial.Size(i)
}

func (i *randomIndex) IndexManifest() (*v1.IndexManifest, error) {
	return i.manifest, nil
}

func (i *randomIndex) RawManifest() ([]byte, error) {
	m, err := i.IndexManifest()
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func (i *randomIndex) Image(h v1.Hash) (v1.Image, error) {
	if img, ok := i.images[h]; ok {
		return img, nil
	}

	return nil, fmt.Errorf("image not found: %v", h)
}

func (i *randomIndex) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	// This is a single level index (for now?).
	return nil, fmt.Errorf("image not found: %v", h)
}
