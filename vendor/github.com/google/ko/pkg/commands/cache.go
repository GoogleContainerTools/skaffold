// Copyright 2023 ko Build Authors All Rights Reserved.
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

package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/ko/pkg/build"
)

type imageCache struct {
	// In memory
	cache sync.Map

	// On disk
	mu sync.Mutex
	p  *layout.Path

	// Over the network
	puller *remote.Puller
}

func newCache(puller *remote.Puller) (*imageCache, error) {
	cache := &imageCache{
		puller: puller,
	}
	if kc := os.Getenv("KOCACHE"); kc != "" {
		path := filepath.Join(kc, "img")
		p, err := layout.FromPath(path)
		if err != nil {
			p, err = layout.Write(path, empty.Index)
			if err != nil {
				return cache, err
			}
		}
		cache.p = &p
	}

	return cache, nil
}

func (i *imageCache) get(ctx context.Context, ref name.Reference, missFunc baseFactory) (build.Result, error) {
	if v, ok := i.cache.Load(ref.String()); ok {
		logs.Debug.Printf("cache hit: %s", ref.String())

		return v.(build.Result), nil
	}

	var (
		once       sync.Once
		missResult build.Result
		missErr    error
	)
	miss := func(ctx context.Context, ref name.Reference) (build.Result, error) {
		once.Do(func() {
			missResult, missErr = missFunc(ctx, ref)
		})

		return missResult, missErr
	}

	if i.p != nil {
		key := ""
		if _, ok := ref.(name.Digest); ok {
			key = ref.Identifier()
		} else {
			logs.Debug.Printf("cache miss due to tag: %s", ref.String())
			result, err := miss(ctx, ref)
			if err != nil {
				return result, err
			}

			dig, err := result.Digest()
			if err != nil {
				return result, err
			}

			key = dig.String()
		}

		// Use a pretty broad lock on the on-disk cache to avoid races.
		i.mu.Lock()
		defer i.mu.Unlock()

		ii, err := i.p.ImageIndex()
		if err != nil {
			return nil, fmt.Errorf("loading cache index: %w", err)
		}

		h, err := v1.NewHash(key)
		if err != nil {
			return nil, err
		}
		descs, err := partial.FindManifests(ii, match.Digests(h))
		if err != nil {
			return nil, err
		}
		if len(descs) != 0 {
			logs.Debug.Printf("cache hit: %s", ref.String())
			desc := descs[0]

			var br build.Result
			if desc.MediaType.IsIndex() {
				idx, err := ii.ImageIndex(h)
				if err != nil {
					return nil, err
				}
				br, err = i.newLazyIndex(ref, idx, missFunc)
				if err != nil {
					return nil, err
				}
			} else {
				img, err := ii.Image(h)
				if err != nil {
					return nil, err
				}
				br, err = i.newLazyImage(ref, img, missFunc)
				if err != nil {
					return nil, err
				}
			}
			i.cache.Store(ref.String(), br)
			return br, nil
		}
	}

	logs.Debug.Printf("cache miss: %s", ref.String())
	result, err := miss(ctx, ref)
	if err != nil {
		return result, err
	}

	if i.p != nil {
		logs.Debug.Printf("cache store: %s", ref.String())

		desc, err := partial.Descriptor(result)
		if err != nil {
			return result, err
		}

		manifest, err := result.RawManifest()
		if err != nil {
			return result, err
		}

		if err := i.p.WriteBlob(desc.Digest, io.NopCloser(bytes.NewReader(manifest))); err != nil {
			return result, err
		}

		if _, ok := result.(v1.ImageIndex); ok {
			result = &lazyIndex{
				ref:      ref,
				desc:     *desc,
				manifest: manifest,
				miss:     missFunc,
				cache:    i,
			}
		} else if img, ok := result.(v1.Image); ok {
			cf, err := img.RawConfigFile()
			if err != nil {
				return result, err
			}

			id, err := img.ConfigName()
			if err != nil {
				return result, err
			}

			if err := i.p.WriteBlob(id, io.NopCloser(bytes.NewReader(cf))); err != nil {
				return result, err
			}

			result = &lazyImage{
				ref:      ref,
				desc:     *desc,
				manifest: manifest,
				config:   cf,
				id:       id,
				miss:     missFunc,
				cache:    i,
			}
		}

		if err := i.p.AppendDescriptor(*desc); err != nil {
			return result, err
		}
	}

	i.cache.Store(ref.String(), result)
	return result, nil
}

func (i *imageCache) newLazyIndex(ref name.Reference, idx v1.ImageIndex, missFunc baseFactory) (*lazyIndex, error) {
	desc, err := partial.Descriptor(idx)
	if err != nil {
		return nil, err
	}

	manifest, err := idx.RawManifest()
	if err != nil {
		return nil, err
	}

	return &lazyIndex{
		ref:      ref,
		desc:     *desc,
		manifest: manifest,
		miss:     missFunc,
		cache:    i,
	}, nil
}

func (i *imageCache) newLazyImage(ref name.Reference, img v1.Image, missFunc baseFactory) (*lazyImage, error) {
	desc, err := partial.Descriptor(img)
	if err != nil {
		return nil, err
	}

	manifest, err := img.RawManifest()
	if err != nil {
		return nil, err
	}

	cf, err := img.RawConfigFile()
	if err != nil {
		return nil, err
	}

	id, err := img.ConfigName()
	if err != nil {
		return nil, err
	}

	return &lazyImage{
		ref:      ref,
		desc:     *desc,
		manifest: manifest,
		config:   cf,
		id:       id,
		miss:     missFunc,
		cache:    i,
	}, nil
}

type lazyIndex struct {
	ref      name.Reference
	desc     v1.Descriptor
	manifest []byte

	miss  baseFactory
	cache *imageCache
}

func (i *lazyIndex) MediaType() (types.MediaType, error) {
	return i.desc.MediaType, nil
}

func (i *lazyIndex) Digest() (v1.Hash, error) {
	return i.desc.Digest, nil
}

func (i *lazyIndex) Size() (int64, error) {
	return i.desc.Size, nil
}

func (i *lazyIndex) IndexManifest() (*v1.IndexManifest, error) {
	return v1.ParseIndexManifest(bytes.NewReader(i.manifest))
}

func (i *lazyIndex) RawManifest() ([]byte, error) {
	return i.manifest, nil
}

func (i *lazyIndex) Image(h v1.Hash) (v1.Image, error) {
	br, err := i.cache.get(context.TODO(), i.ref.Context().Digest(h.String()), i.miss)
	if err != nil {
		return nil, fmt.Errorf("Image(%q): %w", h.String(), err)
	}

	img, ok := br.(v1.Image)
	if !ok {
		return nil, fmt.Errorf("Image(%q) is a type %T", h.String(), br)
	}

	return img, nil
}

func (i *lazyIndex) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	br, err := i.cache.get(context.TODO(), i.ref.Context().Digest(h.String()), i.miss)
	if err != nil {
		return nil, err
	}

	ii, ok := br.(v1.ImageIndex)
	if !ok {
		return nil, fmt.Errorf("ImageIndex(%q) is a type %T", h.String(), br)
	}

	return ii, nil
}

type lazyImage struct {
	ref      name.Reference
	desc     v1.Descriptor
	manifest []byte
	config   []byte
	id       v1.Hash

	miss  baseFactory
	cache *imageCache
}

// Layers returns the ordered collection of filesystem layers that comprise this image.
// The order of the list is oldest/base layer first, and most-recent/top layer last.
func (i *lazyImage) Layers() ([]v1.Layer, error) {
	m, err := i.Manifest()
	if err != nil {
		return nil, err
	}

	layers := make([]v1.Layer, 0, len(m.Layers))
	for _, desc := range m.Layers {
		diffid, err := partial.BlobToDiffID(i, desc.Digest)
		if err != nil {
			return nil, err
		}
		layers = append(layers, &lazyLayer{
			ref:    i.ref.Context().Digest(desc.Digest.String()),
			desc:   desc,
			diffid: diffid,
			miss:   i.miss,
			cache:  i.cache,
		})
	}

	return layers, nil
}

func (i *lazyImage) MediaType() (types.MediaType, error) {
	return i.desc.MediaType, nil
}

func (i *lazyImage) Digest() (v1.Hash, error) {
	return i.desc.Digest, nil
}

func (i *lazyImage) Size() (int64, error) {
	return i.desc.Size, nil
}

func (i *lazyImage) ConfigName() (v1.Hash, error) {
	return i.id, nil
}

func (i *lazyImage) ConfigFile() (*v1.ConfigFile, error) {
	return v1.ParseConfigFile(bytes.NewReader(i.config))
}

func (i *lazyImage) RawConfigFile() ([]byte, error) {
	return i.config, nil
}

func (i *lazyImage) Manifest() (*v1.Manifest, error) {
	return v1.ParseManifest(bytes.NewReader(i.manifest))
}

func (i *lazyImage) RawManifest() ([]byte, error) {
	return i.manifest, nil
}

func (i *lazyImage) LayerByDigest(h v1.Hash) (v1.Layer, error) {
	if h == i.id {
		return partial.ConfigLayer(i)
	}

	layers, err := i.Layers()
	if err != nil {
		return nil, err
	}

	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			return nil, err
		}

		if digest == h {
			return layer, nil
		}
	}

	return nil, fmt.Errorf("could not find layer %q in lazyImage %q", h.String(), i.ref)
}

func (i *lazyImage) LayerByDiffID(h v1.Hash) (v1.Layer, error) {
	d, err := partial.DiffIDToBlob(i, h)
	if err != nil {
		return nil, err
	}

	return i.LayerByDigest(d)
}

type lazyLayer struct {
	ref    name.Digest
	desc   v1.Descriptor
	diffid v1.Hash

	miss  baseFactory
	cache *imageCache
}

func (l *lazyLayer) Digest() (v1.Hash, error) {
	return l.desc.Digest, nil
}

func (l *lazyLayer) DiffID() (v1.Hash, error) {
	return l.diffid, nil
}

func (l *lazyLayer) Size() (int64, error) {
	return l.desc.Size, nil
}

func (l *lazyLayer) MediaType() (types.MediaType, error) {
	return l.desc.MediaType, nil
}

func (l *lazyLayer) Compressed() (io.ReadCloser, error) {
	if rc, err := l.cache.p.Blob(l.desc.Digest); err == nil {
		return rc, nil
	}

	rl, err := l.cache.puller.Layer(context.TODO(), l.ref)
	if err != nil {
		return nil, err
	}

	// Note that we intentionally don't cache this because it will slow down cases where registry has it.
	return rl.Compressed()
}

func (l *lazyLayer) Uncompressed() (io.ReadCloser, error) {
	cl, err := partial.CompressedToLayer(l)
	if err != nil {
		return nil, err
	}
	return cl.Uncompressed()
}
