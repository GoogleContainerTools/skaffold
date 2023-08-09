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

package validate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

// Image validates that img does not violate any invariants of the image format.
func Image(img v1.Image, opt ...Option) error {
	errs := []string{}
	if err := validateLayers(img, opt...); err != nil {
		errs = append(errs, fmt.Sprintf("validating layers: %v", err))
	}

	if err := validateConfig(img); err != nil {
		errs = append(errs, fmt.Sprintf("validating config: %v", err))
	}

	if err := validateManifest(img); err != nil {
		errs = append(errs, fmt.Sprintf("validating manifest: %v", err))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n\n"))
	}
	return nil
}

func validateConfig(img v1.Image) error {
	cn, err := img.ConfigName()
	if err != nil {
		return err
	}

	rc, err := img.RawConfigFile()
	if err != nil {
		return err
	}

	hash, size, err := v1.SHA256(bytes.NewReader(rc))
	if err != nil {
		return err
	}

	m, err := img.Manifest()
	if err != nil {
		return err
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return err
	}

	pcf, err := v1.ParseConfigFile(bytes.NewReader(rc))
	if err != nil {
		return err
	}

	errs := []string{}
	if cn != hash {
		errs = append(errs, fmt.Sprintf("mismatched config digest: ConfigName()=%s, SHA256(RawConfigFile())=%s", cn, hash))
	}

	if want, got := m.Config.Size, size; want != got {
		errs = append(errs, fmt.Sprintf("mismatched config size: Manifest.Config.Size()=%d, len(RawConfigFile())=%d", want, got))
	}

	if diff := cmp.Diff(pcf, cf); diff != "" {
		errs = append(errs, fmt.Sprintf("mismatched config content: (-ParseConfigFile(RawConfigFile()) +ConfigFile()) %s", diff))
	}

	if cf.RootFS.Type != "layers" {
		errs = append(errs, fmt.Sprintf("invalid ConfigFile.RootFS.Type: %q != %q", cf.RootFS.Type, "layers"))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func validateLayers(img v1.Image, opt ...Option) error {
	o := makeOptions(opt...)

	layers, err := img.Layers()
	if err != nil {
		return err
	}

	if o.fast {
		return layersExist(layers)
	}

	digests := []v1.Hash{}
	diffids := []v1.Hash{}
	udiffids := []v1.Hash{}
	sizes := []int64{}
	for i, layer := range layers {
		cl, err := computeLayer(layer)
		if errors.Is(err, io.ErrUnexpectedEOF) {
			// Errored while reading tar content of layer because a header or
			// content section was not the correct length. This is most likely
			// due to an incomplete download or otherwise interrupted process.
			m, err := img.Manifest()
			if err != nil {
				return fmt.Errorf("undersized layer[%d] content", i)
			}
			return fmt.Errorf("undersized layer[%d] content: Manifest.Layers[%d].Size=%d", i, i, m.Layers[i].Size)
		}
		if err != nil {
			return err
		}
		// Compute all of these first before we call Config() and Manifest() to allow
		// for lazy access e.g. for stream.Layer.
		digests = append(digests, cl.digest)
		diffids = append(diffids, cl.diffid)
		udiffids = append(udiffids, cl.uncompressedDiffid)
		sizes = append(sizes, cl.size)
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return err
	}

	m, err := img.Manifest()
	if err != nil {
		return err
	}

	errs := []string{}
	for i, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			return err
		}
		diffid, err := layer.DiffID()
		if err != nil {
			return err
		}
		size, err := layer.Size()
		if err != nil {
			return err
		}
		mediaType, err := layer.MediaType()
		if err != nil {
			return err
		}

		if _, err := img.LayerByDigest(digest); err != nil {
			return err
		}

		if _, err := img.LayerByDiffID(diffid); err != nil {
			return err
		}

		if digest != digests[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] digest: Digest()=%s, SHA256(Compressed())=%s", i, digest, digests[i]))
		}

		if m.Layers[i].Digest != digests[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] digest: Manifest.Layers[%d].Digest=%s, SHA256(Compressed())=%s", i, i, m.Layers[i].Digest, digests[i]))
		}

		if diffid != diffids[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] diffid: DiffID()=%s, SHA256(Gunzip(Compressed()))=%s", i, diffid, diffids[i]))
		}

		if diffid != udiffids[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] diffid: DiffID()=%s, SHA256(Uncompressed())=%s", i, diffid, udiffids[i]))
		}

		if cf.RootFS.DiffIDs[i] != diffids[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] diffid: ConfigFile.RootFS.DiffIDs[%d]=%s, SHA256(Gunzip(Compressed()))=%s", i, i, cf.RootFS.DiffIDs[i], diffids[i]))
		}

		if size != sizes[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] size: Size()=%d, len(Compressed())=%d", i, size, sizes[i]))
		}

		if m.Layers[i].Size != sizes[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] size: Manifest.Layers[%d].Size=%d, len(Compressed())=%d", i, i, m.Layers[i].Size, sizes[i]))
		}

		if m.Layers[i].MediaType != mediaType {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] mediaType: Manifest.Layers[%d].MediaType=%s, layer.MediaType()=%s", i, i, m.Layers[i].MediaType, mediaType))
		}
	}
	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func validateManifest(img v1.Image) error {
	digest, err := img.Digest()
	if err != nil {
		return err
	}

	size, err := img.Size()
	if err != nil {
		return err
	}

	rm, err := img.RawManifest()
	if err != nil {
		return err
	}

	hash, _, err := v1.SHA256(bytes.NewReader(rm))
	if err != nil {
		return err
	}

	m, err := img.Manifest()
	if err != nil {
		return err
	}

	pm, err := v1.ParseManifest(bytes.NewReader(rm))
	if err != nil {
		return err
	}

	errs := []string{}
	if digest != hash {
		errs = append(errs, fmt.Sprintf("mismatched manifest digest: Digest()=%s, SHA256(RawManifest())=%s", digest, hash))
	}

	if diff := cmp.Diff(pm, m); diff != "" {
		errs = append(errs, fmt.Sprintf("mismatched manifest content: (-ParseManifest(RawManifest()) +Manifest()) %s", diff))
	}

	if size != int64(len(rm)) {
		errs = append(errs, fmt.Sprintf("mismatched manifest size: Size()=%d, len(RawManifest())=%d", size, len(rm)))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func layersExist(layers []v1.Layer) error {
	errs := []string{}
	for _, layer := range layers {
		ok, err := partial.Exists(layer)
		if err != nil {
			errs = append(errs, err.Error())
		}
		if !ok {
			errs = append(errs, "layer does not exist")
		}
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}
