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
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
)

func newFuture(work func() (name.Reference, error)) *future {
	// Create a channel on which to send the result.
	ch := make(chan *result)
	// Initiate the actual work, sending its result
	// along the above channel.
	go func() {
		ref, err := work()
		ch <- &result{ref: ref, err: err}
	}()
	// Return a future for the above work.  Callers should
	// call .Get() on this result (as many times as needed).
	// One of these calls will receive the result, store it,
	// and close the channel so that the rest of the callers
	// can consume it.
	return &future{
		promise: ch,
	}
}

type result struct {
	ref name.Reference
	err error
}

type future struct {
	m sync.RWMutex

	result  *result
	promise chan *result
}

// Get blocks on the result of the future.
func (f *future) Get() (name.Reference, error) {
	// Block on the promise of a result until we get one.
	result, ok := <-f.promise
	if ok {
		func() {
			f.m.Lock()
			defer f.m.Unlock()
			// If we got the result, then store it so that
			// others may access it.
			f.result = result
			// Close the promise channel so that others
			// are signaled that the result is available.
			close(f.promise)
		}()
	}

	f.m.RLock()
	defer f.m.RUnlock()

	return f.result.ref, f.result.err
}
