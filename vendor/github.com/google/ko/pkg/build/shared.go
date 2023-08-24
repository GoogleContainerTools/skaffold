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

package build

import (
	"context"
	"sync"
)

// Caching wraps a builder implementation in a layer that shares build results
// for the same inputs using a simple "future" implementation.  Cached results
// may be invalidated by calling Invalidate with the same input passed to Build.
type Caching struct {
	inner Interface

	m       sync.Mutex
	results map[string]*future
}

// Caching implements Interface
var _ Interface = (*Caching)(nil)

// NewCaching wraps the provided build.Interface in an implementation that
// shares build results for a given path until the result has been invalidated.
func NewCaching(inner Interface) (*Caching, error) {
	return &Caching{
		inner:   inner,
		results: make(map[string]*future),
	}, nil
}

// Build implements Interface
func (c *Caching) Build(ctx context.Context, ip string) (Result, error) {
	f := func() *future {
		// Lock the map of futures.
		c.m.Lock()
		defer c.m.Unlock()

		// If a future for "ip" exists, then return it.
		f, ok := c.results[ip]
		if ok {
			return f
		}
		// Otherwise create and record a future for a Build of "ip".
		f = newFuture(func() (Result, error) {
			return c.inner.Build(ctx, ip)
		})
		c.results[ip] = f
		return f
	}()

	return f.Get()
}

// QualifyImport implements Interface
func (c *Caching) QualifyImport(ip string) (string, error) {
	return c.inner.QualifyImport(ip)
}

// IsSupportedReference implements Interface
func (c *Caching) IsSupportedReference(ip string) error {
	return c.inner.IsSupportedReference(ip)
}

// Invalidate removes an import path's cached results.
func (c *Caching) Invalidate(ip string) {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.results, ip)
}
