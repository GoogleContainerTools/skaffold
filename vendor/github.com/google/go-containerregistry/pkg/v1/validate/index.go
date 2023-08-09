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
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/logs"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// Index validates that idx does not violate any invariants of the index format.
func Index(idx v1.ImageIndex, opt ...Option) error {
	errs := []string{}

	if err := validateChildren(idx, opt...); err != nil {
		errs = append(errs, fmt.Sprintf("validating children: %v", err))
	}

	if err := validateIndexManifest(idx); err != nil {
		errs = append(errs, fmt.Sprintf("validating index manifest: %v", err))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n\n"))
	}
	return nil
}

type withLayer interface {
	Layer(v1.Hash) (v1.Layer, error)
}

func validateChildren(idx v1.ImageIndex, opt ...Option) error {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return err
	}

	errs := []string{}
	for i, desc := range manifest.Manifests {
		switch desc.MediaType {
		case types.OCIImageIndex, types.DockerManifestList:
			idx, err := idx.ImageIndex(desc.Digest)
			if err != nil {
				return err
			}
			if err := Index(idx, opt...); err != nil {
				errs = append(errs, fmt.Sprintf("failed to validate index Manifests[%d](%s): %v", i, desc.Digest, err))
			}
			if err := validateMediaType(idx, desc.MediaType); err != nil {
				errs = append(errs, fmt.Sprintf("failed to validate index MediaType[%d](%s): %v", i, desc.Digest, err))
			}
		case types.OCIManifestSchema1, types.DockerManifestSchema2:
			img, err := idx.Image(desc.Digest)
			if err != nil {
				return err
			}
			if err := Image(img, opt...); err != nil {
				errs = append(errs, fmt.Sprintf("failed to validate image Manifests[%d](%s): %v", i, desc.Digest, err))
			}
			if err := validateMediaType(img, desc.MediaType); err != nil {
				errs = append(errs, fmt.Sprintf("failed to validate image MediaType[%d](%s): %v", i, desc.Digest, err))
			}
		default:
			// Workaround for #819.
			if wl, ok := idx.(withLayer); ok {
				layer, err := wl.Layer(desc.Digest)
				if err != nil {
					return fmt.Errorf("failed to get layer Manifests[%d]: %w", i, err)
				}
				if err := Layer(layer, opt...); err != nil {
					lerr := fmt.Sprintf("failed to validate layer Manifests[%d](%s): %v", i, desc.Digest, err)
					if desc.MediaType.IsDistributable() {
						errs = append(errs, lerr)
					} else {
						logs.Warn.Printf("nondistributable layer failure: %v", lerr)
					}
				}
			} else {
				logs.Warn.Printf("Unexpected manifest: %s", desc.MediaType)
			}
		}
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

type withMediaType interface {
	MediaType() (types.MediaType, error)
}

func validateMediaType(i withMediaType, want types.MediaType) error {
	got, err := i.MediaType()
	if err != nil {
		return err
	}
	if want != got {
		return fmt.Errorf("mismatched mediaType: MediaType() = %v != %v", got, want)
	}

	return nil
}

func validateIndexManifest(idx v1.ImageIndex) error {
	digest, err := idx.Digest()
	if err != nil {
		return err
	}

	size, err := idx.Size()
	if err != nil {
		return err
	}

	rm, err := idx.RawManifest()
	if err != nil {
		return err
	}

	hash, _, err := v1.SHA256(bytes.NewReader(rm))
	if err != nil {
		return err
	}

	m, err := idx.IndexManifest()
	if err != nil {
		return err
	}

	pm, err := v1.ParseIndexManifest(bytes.NewReader(rm))
	if err != nil {
		return err
	}

	errs := []string{}
	if digest != hash {
		errs = append(errs, fmt.Sprintf("mismatched manifest digest: Digest()=%s, SHA256(RawManifest())=%s", digest, hash))
	}

	if diff := cmp.Diff(pm, m); diff != "" {
		errs = append(errs, fmt.Sprintf("mismatched manifest content: (-ParseIndexManifest(RawManifest()) +Manifest()) %s", diff))
	}

	if size != int64(len(rm)) {
		errs = append(errs, fmt.Sprintf("mismatched manifest size: Size()=%d, len(RawManifest())=%d", size, len(rm)))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}
