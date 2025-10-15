// Copyright 2025 Google LLC. All Rights Reserved.
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

package fetcher

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// PartialOrFullResource calls the provided function with the provided partial resource size value in order to fetch and return a static resource.
// If p is non-zero, and f returns os.ErrNotExist, this function will try to fetch the corresponding full resource by calling f a second time passing
// zero.
func PartialOrFullResource(ctx context.Context, p uint8, f func(context.Context, uint8) ([]byte, error)) ([]byte, error) {
	sRaw, err := f(ctx, p)
	switch {
	case errors.Is(err, os.ErrNotExist) && p == 0:
		return sRaw, fmt.Errorf("resource not found: %w", err)
	case errors.Is(err, os.ErrNotExist) && p > 0:
		// It could be that the partial resource was removed as the tree has grown and a full resource is now present, so try
		// falling back to that.
		sRaw, err = f(ctx, 0)
		if err != nil {
			return sRaw, fmt.Errorf("neither partial nor full resource found: %w", err)
		}
		return sRaw, nil
	case err != nil:
		return sRaw, fmt.Errorf("failed to fetch resource: %v", err)
	default:
		return sRaw, nil
	}
}
