// Copyright 2018 ko Build Authors All Rights Reserved.
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

package publish

import (
	"context"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/ko/pkg/build"
)

// caching wraps a publisher implementation in a layer that shares publish results
// for the same inputs using a simple "future" implementation.
type caching struct {
	inner Interface

	m       sync.Mutex
	results map[string]*entry
}

// entry holds the last image published and the result of publishing it for a
// particular reference.
type entry struct {
	br build.Result
	f  *future
}

// caching implements Interface
var _ Interface = (*caching)(nil)

// NewCaching wraps the provided publish.Interface in an implementation that
// shares publish results for a given path until the passed image object changes.
func NewCaching(inner Interface) (Interface, error) {
	return &caching{
		inner:   inner,
		results: make(map[string]*entry),
	}, nil
}

// Publish implements Interface
func (c *caching) Publish(ctx context.Context, br build.Result, ref string) (name.Reference, error) {
	f := func() *future {
		// Lock the map of futures.
		c.m.Lock()
		defer c.m.Unlock()

		// If a future for "ref" exists, then return it.
		ent, ok := c.results[ref]
		if ok {
			// If the image matches, then return the same future.
			if ent.br == br {
				return ent.f
			}
		}
		// Otherwise create and record a future for publishing "br" to "ref".
		f := newFuture(func() (name.Reference, error) {
			return c.inner.Publish(ctx, br, ref)
		})
		c.results[ref] = &entry{br: br, f: f}
		return f
	}()

	return f.Get()
}

func (c *caching) Close() error {
	return c.inner.Close()
}
