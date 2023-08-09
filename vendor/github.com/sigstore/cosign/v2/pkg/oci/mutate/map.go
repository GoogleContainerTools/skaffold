//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mutate

import (
	"context"
	"errors"
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sigstore/cosign/v2/pkg/oci"
)

// Fn is the signature of the callback supplied to Map.
// The oci.SignedEntity is either an oci.SignedImageIndex or an oci.SignedImage.
// This callback is called on oci.SignedImageIndex *before* its children are
// processed with a context that returns IsBeforeChildren(ctx) == true.
// If the images within the SignedImageIndex change after the Before pass, then
// the Fn will be invoked again on the new SignedImageIndex with a context
// that returns IsAfterChildren(ctx) == true.
// If the returned entity is nil, it is filtered from the result of Map.
type Fn func(context.Context, oci.SignedEntity) (oci.SignedEntity, error)

// ErrSkipChildren is a special error that may be returned from a Mutator
// to skip processing of an index's child entities.
var ErrSkipChildren = errors.New("skip child entities")

// Map calls `fn` on the signed entity and each of its constituent entities (`SignedImageIndex`
// or `SignedImage`) transitively.
// Any errors returned by an `fn` are returned by `Map`.
func Map(ctx context.Context, parent oci.SignedEntity, fn Fn) (oci.SignedEntity, error) {
	parent, err := fn(before(ctx), parent)
	switch {
	case errors.Is(err, ErrSkipChildren):
		return parent, nil
	case err != nil:
		return nil, err
	case parent == nil:
		// If the function returns nil, it filters it.
		return nil, nil
	}

	sii, ok := parent.(oci.SignedImageIndex)
	if !ok {
		return parent, nil
	}
	im, err := sii.IndexManifest()
	if err != nil {
		return nil, err
	}

	// Track whether any of the child entities change.
	changed := false

	adds := []IndexAddendum{}
	for _, desc := range im.Manifests {
		switch desc.MediaType {
		case types.OCIImageIndex, types.DockerManifestList:
			x, err := sii.SignedImageIndex(desc.Digest)
			if err != nil {
				return nil, err
			}

			se, err := Map(ctx, x, fn)
			if err != nil {
				return nil, err
			} else if se == nil {
				// If the function returns nil, it filters it.
				changed = true
				continue
			}

			changed = changed || (x != se)
			adds = append(adds, IndexAddendum{
				Add: se.(oci.SignedImageIndex), // Must be an image index.
				Descriptor: v1.Descriptor{
					URLs:        desc.URLs,
					MediaType:   desc.MediaType,
					Annotations: desc.Annotations,
					Platform:    desc.Platform,
				},
			})

		case types.OCIManifestSchema1, types.DockerManifestSchema2:
			x, err := sii.SignedImage(desc.Digest)
			if err != nil {
				return nil, err
			}

			se, err := fn(ctx, x)
			if err != nil {
				return nil, err
			} else if se == nil {
				// If the function returns nil, it filters it.
				changed = true
				continue
			}

			changed = changed || (x != se)
			adds = append(adds, IndexAddendum{
				Add: se.(oci.SignedImage), // Must be an image
				Descriptor: v1.Descriptor{
					URLs:        desc.URLs,
					MediaType:   desc.MediaType,
					Annotations: desc.Annotations,
					Platform:    desc.Platform,
				},
			})

		default:
			return nil, fmt.Errorf("unknown mime type: %v", desc.MediaType)
		}
	}

	if !changed {
		return parent, nil
	}

	// Preserve the key attributes from the base IndexManifest.
	e := mutate.IndexMediaType(empty.Index, im.MediaType)
	e = mutate.Annotations(e, im.Annotations).(v1.ImageIndex)

	// Construct a new ImageIndex from the new constituent signed images.
	result := AppendManifests(e, adds...)

	// Since the children changed, give the callback a crack at the new image index.
	return fn(after(ctx), result)
}

// This is used to associate which pass of the Map a particular
// callback is being invoked for.
type mapPassKey struct{}

// before decorates the context such that IsBeforeChildren(ctx) is true.
func before(ctx context.Context) context.Context {
	return context.WithValue(ctx, mapPassKey{}, "before")
}

// after decorates the context such that IsAfterChildren(ctx) is true.
func after(ctx context.Context) context.Context {
	return context.WithValue(ctx, mapPassKey{}, "after")
}

// IsBeforeChildren is true within a Mutator when it is called before the children
// have been processed.
func IsBeforeChildren(ctx context.Context) bool {
	return ctx.Value(mapPassKey{}) == "before"
}

// IsAfterChildren is true within a Mutator when it is called after the children
// have been processed; however, this call is only made if the set of children
// changes since the Before call.
func IsAfterChildren(ctx context.Context) bool {
	return ctx.Value(mapPassKey{}) == "after"
}
