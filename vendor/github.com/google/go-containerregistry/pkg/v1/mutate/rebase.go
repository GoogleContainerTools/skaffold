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
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
)

// Rebase returns a new v1.Image where the oldBase in orig is replaced by newBase.
func Rebase(orig, oldBase, newBase v1.Image) (v1.Image, error) {
	// Verify that oldBase's layers are present in orig, otherwise orig is
	// not based on oldBase at all.
	origLayers, err := orig.Layers()
	if err != nil {
		return nil, fmt.Errorf("failed to get layers for original: %v", err)
	}
	oldBaseLayers, err := oldBase.Layers()
	if err != nil {
		return nil, err
	}
	if len(oldBaseLayers) > len(origLayers) {
		return nil, fmt.Errorf("image %q is not based on %q (too few layers)", orig, oldBase)
	}
	for i, l := range oldBaseLayers {
		oldLayerDigest, err := l.Digest()
		if err != nil {
			return nil, fmt.Errorf("failed to get digest of layer %d of %q: %v", i, oldBase, err)
		}
		origLayerDigest, err := origLayers[i].Digest()
		if err != nil {
			return nil, fmt.Errorf("failed to get digest of layer %d of %q: %v", i, orig, err)
		}
		if oldLayerDigest != origLayerDigest {
			return nil, fmt.Errorf("image %q is not based on %q (layer %d mismatch)", orig, oldBase, i)
		}
	}

	origConfig, err := orig.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get config for original: %v", err)
	}

	// Stitch together an image that contains:
	// - original image's config
	// - new base image's layers + top of original image's layers
	// - new base image's history + top of original image's history
	rebasedImage, err := Config(empty.Image, *origConfig.Config.DeepCopy())
	if err != nil {
		return nil, fmt.Errorf("failed to create empty image with original config: %v", err)
	}
	// Get new base layers and config for history.
	newBaseLayers, err := newBase.Layers()
	if err != nil {
		return nil, fmt.Errorf("could not get new base layers for new base: %v", err)
	}
	newConfig, err := newBase.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("could not get config for new base: %v", err)
	}
	// Add new base layers.
	for i := range newBaseLayers {
		rebasedImage, err = Append(rebasedImage, Addendum{
			Layer:   newBaseLayers[i],
			History: newConfig.History[i],
		})
		if err != nil {
			return nil, fmt.Errorf("failed to append layer %d of new base layers", i)
		}
	}
	// Add original layers above the old base.
	start := len(oldBaseLayers)
	for i := range origLayers[start:] {
		rebasedImage, err = Append(rebasedImage, Addendum{
			Layer:   origLayers[start+i],
			History: origConfig.History[start+i],
		})
		if err != nil {
			return nil, fmt.Errorf("failed to append layer %d of original layers", i)
		}
	}
	return rebasedImage, nil
}
