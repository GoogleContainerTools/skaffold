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

package walk

import (
	"context"

	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
)

// Fn is the signature of the callback supplied to SignedEntity.
// The oci.SignedEntity is either an oci.SignedImageIndex or an oci.SignedImage.
// This callback is called on oci.SignedImageIndex *before* its children.
type Fn func(context.Context, oci.SignedEntity) error

// SignedEntity calls `fn` on the signed entity and each of its constituent entities
// (`SignedImageIndex` or `SignedImage`) transitively.
// Any errors returned by an `fn` are returned by `Walk`.
func SignedEntity(ctx context.Context, parent oci.SignedEntity, fn Fn) error {
	_, err := mutate.Map(ctx, parent, func(ctx context.Context, se oci.SignedEntity) (oci.SignedEntity, error) {
		if err := fn(ctx, se); err != nil {
			return nil, err
		}
		return se, nil
	})
	return err
}
