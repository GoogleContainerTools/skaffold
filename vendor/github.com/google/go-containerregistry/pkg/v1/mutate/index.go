// Copyright 2019 Google LLC All Rights Reserved.
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

package mutate

import (
	"encoding/json"

	"github.com/google/go-containerregistry/pkg/logs"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func computeDescriptor(desc v1.Descriptor, add Appendable) (*v1.Descriptor, error) {
	d, err := add.Digest()
	if err != nil {
		return nil, err
	}
	mt, err := add.MediaType()
	if err != nil {
		return nil, err
	}
	sz, err := add.Size()
	if err != nil {
		return nil, err
	}

	// The IndexAddendum allows overriding These values.
	if desc.Size == 0 {
		desc.Size = sz
	}
	if string(desc.MediaType) == "" {
		desc.MediaType = mt
	}
	if desc.Digest == (v1.Hash{}) {
		desc.Digest = d
	}
	return &desc, nil
}

type index struct {
	base v1.ImageIndex
	adds []IndexAddendum

	computed bool
	manifest *v1.IndexManifest
	imageMap map[v1.Hash]v1.Image
	indexMap map[v1.Hash]v1.ImageIndex
}

var _ v1.ImageIndex = (*index)(nil)

func (i *index) MediaType() (types.MediaType, error) { return i.base.MediaType() }
func (i *index) Size() (int64, error)                { return partial.Size(i) }

func (i *index) compute() error {
	// Don't re-compute if already computed.
	if i.computed {
		return nil
	}

	i.imageMap = make(map[v1.Hash]v1.Image)
	i.indexMap = make(map[v1.Hash]v1.ImageIndex)

	m, err := i.base.IndexManifest()
	if err != nil {
		return err
	}
	manifest := m.DeepCopy()
	manifests := manifest.Manifests
	for _, add := range i.adds {
		desc, err := computeDescriptor(add.Descriptor, add.Add)
		if err != nil {
			return err
		}

		manifests = append(manifests, *desc)
		if idx, ok := add.Add.(v1.ImageIndex); ok {
			i.indexMap[desc.Digest] = idx
		} else if img, ok := add.Add.(v1.Image); ok {
			i.imageMap[desc.Digest] = img
		} else {
			logs.Warn.Printf("Unexpected index addendum: %T", add.Add)
		}
	}
	manifest.Manifests = manifests

	i.manifest = manifest
	i.computed = true
	return nil
}

func (i *index) Image(h v1.Hash) (v1.Image, error) {
	if img, ok := i.imageMap[h]; ok {
		return img, nil
	}
	return i.base.Image(h)
}

func (i *index) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	if idx, ok := i.indexMap[h]; ok {
		return idx, nil
	}
	return i.base.ImageIndex(h)
}

// Digest returns the sha256 of this image's manifest.
func (i *index) Digest() (v1.Hash, error) {
	if err := i.compute(); err != nil {
		return v1.Hash{}, err
	}
	return partial.Digest(i)
}

// Manifest returns this image's Manifest object.
func (i *index) IndexManifest() (*v1.IndexManifest, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return i.manifest, nil
}

// RawManifest returns the serialized bytes of Manifest()
func (i *index) RawManifest() ([]byte, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return json.Marshal(i.manifest)
}
