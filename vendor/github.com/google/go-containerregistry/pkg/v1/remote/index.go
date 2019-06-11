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

package remote

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// remoteIndex accesses an index from a remote registry
type remoteIndex struct {
	fetcher
	manifestLock sync.Mutex // Protects manifest
	manifest     []byte
	mediaType    types.MediaType
}

// Index provides access to a remote index reference, applying functional options
// to the underlying imageOpener before resolving the reference into a v1.ImageIndex.
func Index(ref name.Reference, options ...ImageOption) (v1.ImageIndex, error) {
	acceptable := []types.MediaType{
		types.DockerManifestList,
		types.OCIImageIndex,
	}

	desc, err := get(ref, acceptable, options...)
	if err != nil {
		return nil, err
	}

	return desc.ImageIndex()
}

func (r *remoteIndex) MediaType() (types.MediaType, error) {
	if string(r.mediaType) != "" {
		return r.mediaType, nil
	}
	return types.DockerManifestList, nil
}

func (r *remoteIndex) Digest() (v1.Hash, error) {
	return partial.Digest(r)
}

func (r *remoteIndex) RawManifest() ([]byte, error) {
	r.manifestLock.Lock()
	defer r.manifestLock.Unlock()
	if r.manifest != nil {
		return r.manifest, nil
	}

	// NOTE(jonjohnsonjr): We should never get here because the public entrypoints
	// do type-checking via remote.Descriptor. I've left this here for tests that
	// directly instantiate a remoteIndex.
	acceptable := []types.MediaType{
		types.DockerManifestList,
		types.OCIImageIndex,
	}
	manifest, desc, err := r.fetchManifest(r.Ref, acceptable)
	if err != nil {
		return nil, err
	}

	r.mediaType = desc.MediaType
	r.manifest = manifest
	return r.manifest, nil
}

func (r *remoteIndex) IndexManifest() (*v1.IndexManifest, error) {
	b, err := r.RawManifest()
	if err != nil {
		return nil, err
	}
	return v1.ParseIndexManifest(bytes.NewReader(b))
}

func (r *remoteIndex) Image(h v1.Hash) (v1.Image, error) {
	desc, err := r.childByHash(h)
	if err != nil {
		return nil, err
	}

	// Descriptor.Image will handle coercing nested indexes into an Image.
	return desc.Image()
}

func (r *remoteIndex) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	desc, err := r.childByHash(h)
	if err != nil {
		return nil, err
	}
	return desc.ImageIndex()
}

func (r *remoteIndex) imageByPlatform(platform v1.Platform) (v1.Image, error) {
	desc, err := r.childByPlatform(platform)
	if err != nil {
		return nil, err
	}

	// Descriptor.Image will handle coercing nested indexes into an Image.
	return desc.Image()
}

// This naively matches the first manifest with matching Architecture and OS.
//
// We should probably use this instead:
//	 github.com/containerd/containerd/platforms
//
// But first we'd need to migrate to:
//   github.com/opencontainers/image-spec/specs-go/v1
func (r *remoteIndex) childByPlatform(platform v1.Platform) (*Descriptor, error) {
	index, err := r.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, childDesc := range index.Manifests {
		// If platform is missing from child descriptor, assume it's amd64/linux.
		p := defaultPlatform
		if childDesc.Platform != nil {
			p = *childDesc.Platform
		}

		if platform.Architecture == p.Architecture && platform.OS == p.OS {
			return r.childDescriptor(childDesc, platform)
		}
	}
	return nil, fmt.Errorf("no child with platform %s/%s in index %s", platform.Architecture, platform.OS, r.Ref)
}

func (r *remoteIndex) childByHash(h v1.Hash) (*Descriptor, error) {
	index, err := r.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, childDesc := range index.Manifests {
		if h == childDesc.Digest {
			return r.childDescriptor(childDesc, defaultPlatform)
		}
	}
	return nil, fmt.Errorf("no child with digest %s in index %s", h, r.Ref)
}

func (r *remoteIndex) childRef(h v1.Hash) (name.Reference, error) {
	return name.ParseReference(fmt.Sprintf("%s@%s", r.Ref.Context(), h), name.StrictValidation)
}

// Convert one of this index's child's v1.Descriptor into a remote.Descriptor, with the given platform option.
func (r *remoteIndex) childDescriptor(child v1.Descriptor, platform v1.Platform) (*Descriptor, error) {
	ref, err := r.childRef(child.Digest)
	if err != nil {
		return nil, err
	}
	manifest, desc, err := r.fetchManifest(ref, []types.MediaType{child.MediaType})
	if err != nil {
		return nil, err
	}
	return &Descriptor{
		fetcher: fetcher{
			Ref:    ref,
			Client: r.Client,
		},
		Manifest:   manifest,
		Descriptor: *desc,
		platform:   platform,
	}, nil
}
