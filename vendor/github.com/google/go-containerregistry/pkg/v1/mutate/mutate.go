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

package mutate

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/go-containerregistry/pkg/v1/v1util"
)

const whiteoutPrefix = ".wh."

// Addendum contains layers and history to be appended
// to a base image
type Addendum struct {
	Layer       v1.Layer
	History     v1.History
	URLs        []string
	Annotations map[string]string
}

// AppendLayers applies layers to a base image
func AppendLayers(base v1.Image, layers ...v1.Layer) (v1.Image, error) {
	additions := make([]Addendum, 0, len(layers))
	for _, layer := range layers {
		additions = append(additions, Addendum{Layer: layer})
	}

	return Append(base, additions...)
}

// Append will apply the list of addendums to the base image
func Append(base v1.Image, adds ...Addendum) (v1.Image, error) {
	if len(adds) == 0 {
		return base, nil
	}
	if err := validate(adds); err != nil {
		return nil, err
	}

	return &image{
		base: base,
		adds: adds,
	}, nil
}

// Config mutates the provided v1.Image to have the provided v1.Config
func Config(base v1.Image, cfg v1.Config) (v1.Image, error) {
	cf, err := base.ConfigFile()
	if err != nil {
		return nil, err
	}

	cf.Config = cfg
	// Downstream tooling expects these to match.
	cf.ContainerConfig = cfg

	return ConfigFile(base, cf)
}

// ConfigFile mutates the provided v1.Image to have the provided v1.ConfigFile
func ConfigFile(base v1.Image, cfg *v1.ConfigFile) (v1.Image, error) {
	m, err := base.Manifest()
	if err != nil {
		return nil, err
	}

	image := &image{
		base:       base,
		manifest:   m.DeepCopy(),
		configFile: cfg,
	}

	return image, nil
}

// CreatedAt mutates the provided v1.Image to have the provided v1.Time
func CreatedAt(base v1.Image, created v1.Time) (v1.Image, error) {
	cf, err := base.ConfigFile()
	if err != nil {
		return nil, err
	}

	cfg := cf.DeepCopy()
	cfg.Created = created

	return ConfigFile(base, cfg)
}

type image struct {
	base v1.Image
	adds []Addendum

	computed   bool
	configFile *v1.ConfigFile
	manifest   *v1.Manifest
	diffIDMap  map[v1.Hash]v1.Layer
	digestMap  map[v1.Hash]v1.Layer
}

var _ v1.Image = (*image)(nil)

func (i *image) MediaType() (types.MediaType, error) { return i.base.MediaType() }

func (i *image) compute() error {
	// Don't re-compute if already computed.
	if i.computed {
		return nil
	}
	var configFile *v1.ConfigFile
	if i.configFile != nil {
		configFile = i.configFile
	} else {
		cf, err := i.base.ConfigFile()
		if err != nil {
			return err
		}
		configFile = cf.DeepCopy()
	}
	diffIDs := configFile.RootFS.DiffIDs
	history := configFile.History

	diffIDMap := make(map[v1.Hash]v1.Layer)
	digestMap := make(map[v1.Hash]v1.Layer)

	for _, add := range i.adds {
		diffID, err := add.Layer.DiffID()
		if err != nil {
			return err
		}
		diffIDs = append(diffIDs, diffID)
		history = append(history, add.History)
		diffIDMap[diffID] = add.Layer
	}

	m, err := i.base.Manifest()
	if err != nil {
		return err
	}
	manifest := m.DeepCopy()
	manifestLayers := manifest.Layers
	for _, add := range i.adds {
		d := v1.Descriptor{}
		var err error

		if d.Size, err = add.Layer.Size(); err != nil {
			return err
		}

		if d.Digest, err = add.Layer.Digest(); err != nil {
			return err
		}

		if d.MediaType, err = add.Layer.MediaType(); err != nil {
			return err
		}

		d.Annotations = add.Annotations
		d.URLs = add.URLs

		manifestLayers = append(manifestLayers, d)
		digestMap[d.Digest] = add.Layer
	}

	configFile.RootFS.DiffIDs = diffIDs
	configFile.History = history

	manifest.Layers = manifestLayers

	rcfg, err := json.Marshal(configFile)
	if err != nil {
		return err
	}
	d, sz, err := v1.SHA256(bytes.NewBuffer(rcfg))
	if err != nil {
		return err
	}
	manifest.Config.Digest = d
	manifest.Config.Size = sz

	i.configFile = configFile
	i.manifest = manifest
	i.diffIDMap = diffIDMap
	i.digestMap = digestMap
	i.computed = true
	return nil
}

// Layers returns the ordered collection of filesystem layers that comprise this image.
// The order of the list is oldest/base layer first, and most-recent/top layer last.
func (i *image) Layers() ([]v1.Layer, error) {
	if err := i.compute(); err == stream.ErrNotComputed {
		// Image contains a streamable layer which has not yet been
		// consumed. Just return the layers we have in case the caller
		// is going to consume the layers.
		layers, err := i.base.Layers()
		if err != nil {
			return nil, err
		}
		for _, add := range i.adds {
			layers = append(layers, add.Layer)
		}
		return layers, nil
	} else if err != nil {
		return nil, err
	}

	diffIDs, err := partial.DiffIDs(i)
	if err != nil {
		return nil, err
	}
	ls := make([]v1.Layer, 0, len(diffIDs))
	for _, h := range diffIDs {
		l, err := i.LayerByDiffID(h)
		if err != nil {
			return nil, err
		}
		ls = append(ls, l)
	}
	return ls, nil
}

// ConfigName returns the hash of the image's config file.
func (i *image) ConfigName() (v1.Hash, error) {
	if err := i.compute(); err != nil {
		return v1.Hash{}, err
	}
	return partial.ConfigName(i)
}

// ConfigFile returns this image's config file.
func (i *image) ConfigFile() (*v1.ConfigFile, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return i.configFile, nil
}

// RawConfigFile returns the serialized bytes of ConfigFile()
func (i *image) RawConfigFile() ([]byte, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return json.Marshal(i.configFile)
}

// Digest returns the sha256 of this image's manifest.
func (i *image) Digest() (v1.Hash, error) {
	if err := i.compute(); err != nil {
		return v1.Hash{}, err
	}
	return partial.Digest(i)
}

// Manifest returns this image's Manifest object.
func (i *image) Manifest() (*v1.Manifest, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return i.manifest, nil
}

// RawManifest returns the serialized bytes of Manifest()
func (i *image) RawManifest() ([]byte, error) {
	if err := i.compute(); err != nil {
		return nil, err
	}
	return json.Marshal(i.manifest)
}

// LayerByDigest returns a Layer for interacting with a particular layer of
// the image, looking it up by "digest" (the compressed hash).
func (i *image) LayerByDigest(h v1.Hash) (v1.Layer, error) {
	if cn, err := i.ConfigName(); err != nil {
		return nil, err
	} else if h == cn {
		return partial.ConfigLayer(i)
	}
	if layer, ok := i.digestMap[h]; ok {
		return layer, nil
	}
	return i.base.LayerByDigest(h)
}

// LayerByDiffID is an analog to LayerByDigest, looking up by "diff id"
// (the uncompressed hash).
func (i *image) LayerByDiffID(h v1.Hash) (v1.Layer, error) {
	if layer, ok := i.diffIDMap[h]; ok {
		return layer, nil
	}
	return i.base.LayerByDiffID(h)
}

func validate(adds []Addendum) error {
	for _, add := range adds {
		if add.Layer == nil {
			return errors.New("Unable to add a nil layer to the image")
		}
	}
	return nil
}

// Extract takes an image and returns an io.ReadCloser containing the image's
// flattened filesystem.
//
// Callers can read the filesystem contents by passing the reader to
// tar.NewReader, or io.Copy it directly to some output.
//
// If a caller doesn't read the full contents, they should Close it to free up
// resources used during extraction.
//
// Adapted from https://github.com/google/containerregistry/blob/master/client/v2_2/docker_image_.py#L731
func Extract(img v1.Image) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		// Close the writer with any errors encountered during
		// extraction. These errors will be returned by the reader end
		// on subsequent reads. If err == nil, the reader will return
		// EOF.
		pw.CloseWithError(extract(img, pw))
	}()

	return pr
}

func extract(img v1.Image, w io.Writer) error {
	tarWriter := tar.NewWriter(w)
	defer tarWriter.Close()

	fileMap := map[string]bool{}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("retrieving image layers: %v", err)
	}
	// we iterate through the layers in reverse order because it makes handling
	// whiteout layers more efficient, since we can just keep track of the removed
	// files as we see .wh. layers and ignore those in previous layers.
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]
		layerReader, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("reading layer contents: %v", err)
		}
		tarReader := tar.NewReader(layerReader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("reading tar: %v", err)
			}

			basename := filepath.Base(header.Name)
			dirname := filepath.Dir(header.Name)
			tombstone := strings.HasPrefix(basename, whiteoutPrefix)
			if tombstone {
				basename = basename[len(whiteoutPrefix):]
			}

			// check if we have seen value before
			// if we're checking a directory, don't filepath.Join names
			var name string
			if header.Typeflag == tar.TypeDir {
				name = header.Name
			} else {
				name = filepath.Join(dirname, basename)
			}

			if _, ok := fileMap[name]; ok {
				continue
			}

			// check for a whited out parent directory
			if inWhiteoutDir(fileMap, name) {
				continue
			}

			// mark file as handled. non-directory implicitly tombstones
			// any entries with a matching (or child) name
			fileMap[name] = tombstone || !(header.Typeflag == tar.TypeDir)
			if !tombstone {
				tarWriter.WriteHeader(header)
				if header.Size > 0 {
					if _, err := io.Copy(tarWriter, tarReader); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func inWhiteoutDir(fileMap map[string]bool, file string) bool {
	for {
		if file == "" {
			break
		}
		dirname := filepath.Dir(file)
		if file == dirname {
			break
		}
		if val, ok := fileMap[dirname]; ok && val {
			return true
		}
		file = dirname
	}
	return false
}

// Time sets all timestamps in an image to the given timestamp.
func Time(img v1.Image, t time.Time) (v1.Image, error) {
	newImage := empty.Image

	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("Error getting image layers: %v", err)
	}

	// Strip away all timestamps from layers
	var newLayers []v1.Layer
	for _, layer := range layers {
		newLayer, err := layerTime(layer, t)
		if err != nil {
			return nil, fmt.Errorf("Error setting layer times: %v", err)
		}
		newLayers = append(newLayers, newLayer)
	}

	newImage, err = AppendLayers(newImage, newLayers...)
	if err != nil {
		return nil, fmt.Errorf("Error appending layers: %v", err)
	}

	ocf, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("Error getting original config file: %v", err)
	}

	cf, err := newImage.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("Error setting config file: %v", err)
	}

	cfg := cf.DeepCopy()

	// Copy basic config over
	cfg.Config = ocf.Config
	cfg.ContainerConfig = ocf.Config // Downstream tooling expects these to match.

	// Strip away timestamps from the config file
	cfg.Created = v1.Time{Time: t}

	for _, h := range cfg.History {
		h.Created = v1.Time{Time: t}
	}

	return ConfigFile(newImage, cfg)
}

func layerTime(layer v1.Layer, t time.Time) (v1.Layer, error) {
	layerReader, err := layer.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("Error getting layer: %v", err)
	}
	w := new(bytes.Buffer)
	tarWriter := tar.NewWriter(w)
	defer tarWriter.Close()

	tarReader := tar.NewReader(layerReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading layer: %v", err)
		}

		header.ModTime = t
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("Error writing tar header: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			if _, err = io.Copy(tarWriter, tarReader); err != nil {
				return nil, fmt.Errorf("Error writing layer file: %v", err)
			}
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	b := w.Bytes()
	// gzip the contents, then create the layer
	opener := func() (io.ReadCloser, error) {
		return v1util.GzipReadCloser(ioutil.NopCloser(bytes.NewReader(b))), nil
	}
	layer, err = tarball.LayerFromOpener(opener)
	if err != nil {
		return nil, fmt.Errorf("Error creating layer: %v", err)
	}

	return layer, nil
}

// Canonical is a helper function to combine Time and configFile
// to remove any randomness during a docker build.
func Canonical(img v1.Image) (v1.Image, error) {
	// Set all timestamps to 0
	created := time.Time{}
	img, err := Time(img, created)
	if err != nil {
		return nil, err
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	// Get rid of host-dependent random config
	cfg := cf.DeepCopy()

	cfg.Container = ""
	cfg.Config.Hostname = ""
	cfg.ContainerConfig.Hostname = ""
	cfg.DockerVersion = ""

	return ConfigFile(img, cfg)
}
