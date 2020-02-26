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
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
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

// AppendLayers applies layers to a base image.
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

// Appendable is an interface that represents something that can be appended
// to an ImageIndex. We need to be able to construct a v1.Descriptor in order
// to append something, and this is the minimum required information for that.
type Appendable interface {
	MediaType() (types.MediaType, error)
	Digest() (v1.Hash, error)
	Size() (int64, error)
}

// IndexAddendum represents an appendable thing and all the properties that
// we may want to override in the resulting v1.Descriptor.
type IndexAddendum struct {
	Add Appendable
	v1.Descriptor
}

// AppendManifests appends a manifest to the ImageIndex.
func AppendManifests(base v1.ImageIndex, adds ...IndexAddendum) v1.ImageIndex {
	return &index{
		base: base,
		adds: adds,
	}
}

// Config mutates the provided v1.Image to have the provided v1.Config
func Config(base v1.Image, cfg v1.Config) (v1.Image, error) {
	cf, err := base.ConfigFile()
	if err != nil {
		return nil, err
	}

	cf.Config = cfg

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

// Extract takes an image and returns an io.ReadCloser containing the image's
// flattened filesystem.
//
// Callers can read the filesystem contents by passing the reader to
// tar.NewReader, or io.Copy it directly to some output.
//
// If a caller doesn't read the full contents, they should Close it to free up
// resources used during extraction.
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

// Adapted from https://github.com/google/containerregistry/blob/da03b395ccdc4e149e34fbb540483efce962dc64/client/v2_2/docker_image_.py#L816
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
		return nil, fmt.Errorf("getting image layers: %v", err)
	}

	// Strip away all timestamps from layers
	var newLayers []v1.Layer
	for _, layer := range layers {
		newLayer, err := layerTime(layer, t)
		if err != nil {
			return nil, fmt.Errorf("setting layer times: %v", err)
		}
		newLayers = append(newLayers, newLayer)
	}

	newImage, err = AppendLayers(newImage, newLayers...)
	if err != nil {
		return nil, fmt.Errorf("appending layers: %v", err)
	}

	ocf, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("getting original config file: %v", err)
	}

	cf, err := newImage.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("setting config file: %v", err)
	}

	cfg := cf.DeepCopy()

	// Copy basic config over
	cfg.Config = ocf.Config

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
		return nil, fmt.Errorf("getting layer: %v", err)
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
			return nil, fmt.Errorf("reading layer: %v", err)
		}

		header.ModTime = t
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("writing tar header: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			if _, err = io.Copy(tarWriter, tarReader); err != nil {
				return nil, fmt.Errorf("writing layer file: %v", err)
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
		return nil, fmt.Errorf("creating layer: %v", err)
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
	cfg.DockerVersion = ""

	return ConfigFile(img, cfg)
}

// MediaType modifies the MediaType() of the given image.
func MediaType(img v1.Image, mt types.MediaType) v1.Image {
	return &image{
		base:      img,
		mediaType: &mt,
	}
}

// IndexMediaType modifies the MediaType() of the given index.
func IndexMediaType(idx v1.ImageIndex, mt types.MediaType) v1.ImageIndex {
	return &index{
		base:      idx,
		mediaType: &mt,
	}
}
