/*
Copyright 2020 The Skaffold Authors

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

package util

import (
	"sync"
)

// SyncStore exports a single method `Exec` to ensure single execution of a function and share the result between all callers of the function.
type SyncStore struct {
	oncePerKey *sync.Map
	results    *sync.Map
}

// Exec executes the function f if and only if it's being called the first time for a specific key.
// If it's called multiple times for the same key only the first call will execute and store the result of f.
// All other calls will be blocked until the running instance of f returns and all of them receive the same result.
func (o *SyncStore) Exec(key interface{}, f func() interface{}) interface{} {
	once, _ := o.oncePerKey.LoadOrStore(key, new(sync.Once))
	once.(*sync.Once).Do(func() {
		res := f()
		o.results.Store(key, res)
	})

	val, _ := o.results.Load(key)
	return val
}

// NewSyncStore returns a new instance of `SyncStore`
func NewSyncStore() *SyncStore {
	return &SyncStore{
		oncePerKey: new(sync.Map),
		results:    new(sync.Map),
	}
}
