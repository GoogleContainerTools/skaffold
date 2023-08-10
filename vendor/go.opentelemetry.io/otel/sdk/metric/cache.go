// Copyright The OpenTelemetry Authors
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

package metric // import "go.opentelemetry.io/otel/sdk/metric"

import (
	"sync"

	"go.opentelemetry.io/otel/sdk/metric/internal"
)

// cache is a locking storage used to quickly return already computed values.
//
// The zero value of a cache is empty and ready to use.
//
// A cache must not be copied after first use.
//
// All methods of a cache are safe to call concurrently.
type cache[K comparable, V any] struct {
	sync.Mutex
	data map[K]V
}

// Lookup returns the value stored in the cache with the accociated key if it
// exists. Otherwise, f is called and its returned value is set in the cache
// for key and returned.
//
// Lookup is safe to call concurrently. It will hold the cache lock, so f
// should not block excessively.
func (c *cache[K, V]) Lookup(key K, f func() V) V {
	c.Lock()
	defer c.Unlock()

	if c.data == nil {
		val := f()
		c.data = map[K]V{key: val}
		return val
	}
	if v, ok := c.data[key]; ok {
		return v
	}
	val := f()
	c.data[key] = val
	return val
}

// instrumentCache is a cache of instruments. It is scoped at the Meter level
// along with a number type. Meaning all instruments it contains need to belong
// to the same instrumentation.Scope (implicitly) and number type (explicitly).
type instrumentCache[N int64 | float64] struct {
	// aggregators is used to ensure duplicate creations of the same instrument
	// return the same instance of that instrument's aggregator.
	aggregators *cache[instrumentID, aggVal[N]]
	// views is used to ensure if instruments with the same name are created,
	// but do not have the same identifying properties, a warning is logged.
	views *cache[string, instrumentID]
}

// newInstrumentCache returns a new instrumentCache that uses ac as the
// underlying cache for aggregators and vc as the cache for views. If ac or vc
// are nil, a new empty cache will be used.
func newInstrumentCache[N int64 | float64](ac *cache[instrumentID, aggVal[N]], vc *cache[string, instrumentID]) instrumentCache[N] {
	if ac == nil {
		ac = &cache[instrumentID, aggVal[N]]{}
	}
	if vc == nil {
		vc = &cache[string, instrumentID]{}
	}
	return instrumentCache[N]{aggregators: ac, views: vc}
}

// LookupAggregator returns the Aggregator and error for a cached instrument if
// it exist in the cache. Otherwise, f is called and its returned value is set
// in the cache and returned.
//
// LookupAggregator is safe to call concurrently.
func (c instrumentCache[N]) LookupAggregator(id instrumentID, f func() (internal.Aggregator[N], error)) (agg internal.Aggregator[N], err error) {
	v := c.aggregators.Lookup(id, func() aggVal[N] {
		a, err := f()
		return aggVal[N]{Aggregator: a, Err: err}
	})
	return v.Aggregator, v.Err
}

// aggVal is the cached value of an instrumentCache's aggregators cache.
type aggVal[N int64 | float64] struct {
	Aggregator internal.Aggregator[N]
	Err        error
}

// Unique returns if id is unique or a duplicate instrument. If an instrument
// with the same name has already been created, that instrumentID will be
// returned along with false. Otherwise, id is returned with true.
//
// Unique is safe to call concurrently.
func (c instrumentCache[N]) Unique(id instrumentID) (instrumentID, bool) {
	got := c.views.Lookup(id.Name, func() instrumentID { return id })
	return got, id == got
}
