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

package layout

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"golang.org/x/sync/errgroup"
)

var layoutFile = `{
    "imageLayoutVersion": "1.0.0"
}`

// AppendImage writes a v1.Image to the Path and updates
// the index.json to reference it.
func (l Path) AppendImage(img v1.Image, options ...Option) error {
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
		MediaType: mt,
		Size:      int64(len(manifest)),
		Digest:    d,
	}

	for _, opt := range options {
		if err := opt(&desc); err != nil {
			return err
		}
	}

	return l.AppendDescriptor(desc)
}

// AppendIndex writes a v1.ImageIndex to the Path and updates
// the index.json to reference it.
func (l Path) AppendIndex(ii v1.ImageIndex, options ...Option) error {
	if err := l.writeIndex(ii); err != nil {
		return err
	}

	mt, err := ii.MediaType()
	if err != nil {
		return err
	}

	d, err := ii.Digest()
	if err != nil {
		return err
	}

	manifest, err := ii.RawManifest()
	if err != nil {
		return err
	}

	desc := v1.Descriptor{
		MediaType: mt,
		Size:      int64(len(manifest)),
		Digest:    d,
	}

	for _, opt := range options {
		if err := opt(&desc); err != nil {
			return err
		}
	}

	return l.AppendDescriptor(desc)
}

// AppendDescriptor adds a descriptor to the index.json of the Path.
func (l Path) AppendDescriptor(desc v1.Descriptor) error {
	ii, err := l.ImageIndex()
	if err != nil {
		return err
	}

	index, err := ii.IndexManifest()
	if err != nil {
		return err
	}

	index.Manifests = append(index.Manifests, desc)

	rawIndex, err := json.MarshalIndent(index, "", "   ")
	if err != nil {
		return err
	}

	return l.WriteFile("index.json", rawIndex, os.ModePerm)
}

// WriteFile write a file with arbitrary data at an arbitrary location in a v1
// layout. Used mostly internally to write files like "oci-layout" and
// "index.json", also can be used to write other arbitrary files. Do *not* use
// this to write blobs. Use only WriteBlob() for that.
func (l Path) WriteFile(name string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(l.path(), os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	return ioutil.WriteFile(l.path(name), data, perm)

}

// WriteBlob copies a file to the blobs/ directory in the Path from the given ReadCloser at
// blobs/{hash.Algorithm}/{hash.Hex}.
func (l Path) WriteBlob(hash v1.Hash, r io.ReadCloser) error {
	dir := l.path("blobs", hash.Algorithm)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	file := filepath.Join(dir, hash.Hex)
	if _, err := os.Stat(file); err == nil {
		// Blob already exists, that's fine.
		return nil
	}
	w, err := os.Create(file)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}

// TODO: A streaming version of WriteBlob so we don't have to know the hash
// before we write it.

// TODO: For streaming layers we should write to a tmp file then Rename to the
// final digest.
func (l Path) writeLayer(layer v1.Layer) error {
	d, err := layer.Digest()
	if err != nil {
		return err
	}

	r, err := layer.Compressed()
	if err != nil {
		return err
	}

	return l.WriteBlob(d, r)
}

func (l Path) writeImage(img v1.Image) error {
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

	// Write the config.
	cfgName, err := img.ConfigName()
	if err != nil {
		return err
	}
	cfgBlob, err := img.RawConfigFile()
	if err != nil {
		return err
	}
	if err := l.WriteBlob(cfgName, ioutil.NopCloser(bytes.NewReader(cfgBlob))); err != nil {
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

	return l.WriteBlob(d, ioutil.NopCloser(bytes.NewReader(manifest)))
}

func (l Path) writeIndexToFile(indexFile string, ii v1.ImageIndex) error {
	index, err := ii.IndexManifest()
	if err != nil {
		return err
	}

	// Walk the descriptors and write any v1.Image or v1.ImageIndex that we find.
	// If we come across something we don't expect, just write it as a blob.
	for _, desc := range index.Manifests {
		switch desc.MediaType {
		case types.OCIImageIndex, types.DockerManifestList:
			ii, err := ii.ImageIndex(desc.Digest)
			if err != nil {
				return err
			}
			if err := l.writeIndex(ii); err != nil {
				return err
			}
		case types.OCIManifestSchema1, types.DockerManifestSchema2:
			img, err := ii.Image(desc.Digest)
			if err != nil {
				return err
			}
			if err := l.writeImage(img); err != nil {
				return err
			}
		default:
			// TODO: The layout could reference arbitrary things, which we should
			// probably just pass through.
		}
	}

	rawIndex, err := ii.RawManifest()
	if err != nil {
		return err
	}

	return l.WriteFile(indexFile, rawIndex, os.ModePerm)
}

func (l Path) writeIndex(ii v1.ImageIndex) error {
	// Always just write oci-layout file, since it's small.
	if err := l.WriteFile("oci-layout", []byte(layoutFile), os.ModePerm); err != nil {
		return err
	}

	h, err := ii.Digest()
	if err != nil {
		return err
	}

	indexFile := filepath.Join("blobs", h.Algorithm, h.Hex)
	return l.writeIndexToFile(indexFile, ii)

}

// Write constructs a Path at path from an ImageIndex.
//
// The contents are written in the following format:
// At the top level, there is:
//   One oci-layout file containing the version of this image-layout.
//   One index.json file listing descriptors for the contained images.
// Under blobs/, there is, for each image:
//   One file for each layer, named after the layer's SHA.
//   One file for each config blob, named after its SHA.
//   One file for each manifest blob, named after its SHA.
func Write(path string, ii v1.ImageIndex) (Path, error) {
	lp := Path(path)
	// Always just write oci-layout file, since it's small.
	if err := lp.WriteFile("oci-layout", []byte(layoutFile), os.ModePerm); err != nil {
		return "", err
	}

	// TODO create blobs/ in case there is a blobs file which would prevent the directory from being created

	return lp, lp.writeIndexToFile("index.json", ii)
}
