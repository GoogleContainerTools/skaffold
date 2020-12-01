/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package duck

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// CachedInformerFactory implements InformerFactory by delegating to another
// InformerFactory, but memoizing the results.
type CachedInformerFactory struct {
	Delegate InformerFactory

	m     sync.Mutex
	cache map[schema.GroupVersionResource]*informerCache
}

// Check that CachedInformerFactory implements InformerFactory.
var _ InformerFactory = (*CachedInformerFactory)(nil)

// Get implements InformerFactory.
func (cif *CachedInformerFactory) Get(ctx context.Context, gvr schema.GroupVersionResource) (cache.SharedIndexInformer, cache.GenericLister, error) {
	cif.m.Lock()

	if cif.cache == nil {
		cif.cache = make(map[schema.GroupVersionResource]*informerCache)
	}

	ic, ok := cif.cache[gvr]
	if !ok {
		ic = &informerCache{}
		ic.init = func() {
			ic.Lock()
			defer ic.Unlock()

			// double-checked lock to ensure we call the Delegate
			// only once even if multiple goroutines end up inside
			// init() simultaneously
			if ic.hasInformer() {
				return
			}

			ic.inf, ic.lister, ic.err = cif.Delegate.Get(ctx, gvr)
		}
		cif.cache[gvr] = ic
	}

	// If this were done via "defer", then TestDifferentGVRs will fail.
	cif.m.Unlock()

	// The call to the delegate could be slow because it syncs informers, so do
	// this outside of the main lock.
	return ic.Get()
}

type informerCache struct {
	sync.RWMutex

	init func()

	inf    cache.SharedIndexInformer
	lister cache.GenericLister
	err    error
}

// Get returns the cached informer. If it does not yet exist, we first try to
// acquire one by executing the cache's init function.
func (ic *informerCache) Get() (cache.SharedIndexInformer, cache.GenericLister, error) {
	if !ic.initialized() {
		ic.init()
	}
	return ic.inf, ic.lister, ic.err
}

func (ic *informerCache) initialized() bool {
	ic.RLock()
	defer ic.RUnlock()
	return ic.hasInformer()
}

func (ic *informerCache) hasInformer() bool {
	return ic.inf != nil && ic.lister != nil
}
