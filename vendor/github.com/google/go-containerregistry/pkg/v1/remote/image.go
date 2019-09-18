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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/go-containerregistry/pkg/v1/v1util"
)

// remoteImage accesses an image from a remote registry
type remoteImage struct {
	fetcher
	manifestLock sync.Mutex // Protects manifest
	manifest     []byte
	configLock   sync.Mutex // Protects config
	config       []byte
	mediaType    types.MediaType
}

var _ partial.CompressedImageCore = (*remoteImage)(nil)

// Image provides access to a remote image reference.
func Image(ref name.Reference, options ...Option) (v1.Image, error) {
	acceptable := []types.MediaType{
		types.DockerManifestSchema2,
		types.OCIManifestSchema1,
		// We resolve these to images later.
		types.DockerManifestList,
		types.OCIImageIndex,
	}

	desc, err := get(ref, acceptable, options...)
	if err != nil {
		return nil, err
	}

	return desc.Image()
}

func (r *remoteImage) MediaType() (types.MediaType, error) {
	if string(r.mediaType) != "" {
		return r.mediaType, nil
	}
	return types.DockerManifestSchema2, nil
}

func (r *remoteImage) RawManifest() ([]byte, error) {
	r.manifestLock.Lock()
	defer r.manifestLock.Unlock()
	if r.manifest != nil {
		return r.manifest, nil
	}

	// NOTE(jonjohnsonjr): We should never get here because the public entrypoints
	// do type-checking via remote.Descriptor. I've left this here for tests that
	// directly instantiate a remoteImage.
	acceptable := []types.MediaType{
		types.DockerManifestSchema2,
		types.OCIManifestSchema1,
	}
	manifest, desc, err := r.fetchManifest(r.Ref, acceptable)
	if err != nil {
		return nil, err
	}

	r.mediaType = desc.MediaType
	r.manifest = manifest
	return r.manifest, nil
}

func (r *remoteImage) RawConfigFile() ([]byte, error) {
	r.configLock.Lock()
	defer r.configLock.Unlock()
	if r.config != nil {
		return r.config, nil
	}

	m, err := partial.Manifest(r)
	if err != nil {
		return nil, err
	}

	cl, err := r.LayerByDigest(m.Config.Digest)
	if err != nil {
		return nil, err
	}
	body, err := cl.Compressed()
	if err != nil {
		return nil, err
	}
	defer body.Close()

	r.config, err = ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return r.config, nil
}

// remoteLayer implements partial.CompressedLayer
type remoteLayer struct {
	ri     *remoteImage
	digest v1.Hash
}

// Digest implements partial.CompressedLayer
func (rl *remoteLayer) Digest() (v1.Hash, error) {
	return rl.digest, nil
}

// Compressed implements partial.CompressedLayer
func (rl *remoteLayer) Compressed() (io.ReadCloser, error) {
	u := rl.ri.url("blobs", rl.digest.String())
	resp, err := rl.ri.Client.Get(u.String())
	if err != nil {
		return nil, err
	}

	if err := transport.CheckError(resp, http.StatusOK); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return v1util.VerifyReadCloser(resp.Body, rl.digest)
}

// Manifest implements partial.WithManifest so that we can use partial.BlobSize below.
func (rl *remoteLayer) Manifest() (*v1.Manifest, error) {
	return partial.Manifest(rl.ri)
}

// MediaType implements v1.Layer
func (rl *remoteLayer) MediaType() (types.MediaType, error) {
	m, err := rl.Manifest()
	if err != nil {
		return "", err
	}

	for _, layer := range m.Layers {
		if layer.Digest == rl.digest {
			return layer.MediaType, nil
		}
	}

	return "", fmt.Errorf("unable to find layer with digest: %v", rl.digest)
}

// Size implements partial.CompressedLayer
func (rl *remoteLayer) Size() (int64, error) {
	// Look up the size of this digest in the manifest to avoid a request.
	return partial.BlobSize(rl, rl.digest)
}

// ConfigFile implements partial.WithManifestAndConfigFile so that we can use partial.BlobToDiffID below.
func (rl *remoteLayer) ConfigFile() (*v1.ConfigFile, error) {
	return partial.ConfigFile(rl.ri)
}

// DiffID implements partial.WithDiffID so that we don't recompute a DiffID that we already have
// available in our ConfigFile.
func (rl *remoteLayer) DiffID() (v1.Hash, error) {
	return partial.BlobToDiffID(rl, rl.digest)
}

// LayerByDigest implements partial.CompressedLayer
func (r *remoteImage) LayerByDigest(h v1.Hash) (partial.CompressedLayer, error) {
	return &remoteLayer{
		ri:     r,
		digest: h,
	}, nil
}
