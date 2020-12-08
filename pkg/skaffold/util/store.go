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
	"fmt"
	"sync"

	"github.com/golang/groupcache/singleflight"
)

// SyncStore exports a single method `Exec` to ensure single execution of a function
// and share the result between all callers of the function.
type SyncStore struct {
	sf      singleflight.Group
	results sync.Map
}

// Exec executes the function f if and only if it's being called the first time for a specific key.
// If it's called multiple times for the same key only the first call will execute and store the result of f.
// All other calls will be blocked until the running instance of f returns and all of them receive the same result.
func (o *SyncStore) Exec(key string, f func() interface{}) interface{} {
	val, err := o.sf.Do(key, func() (_ interface{}, err error) {
		// trap any runtime error due to synchronization issues.
		defer func() {
			if rErr := recover(); rErr != nil {
				err = retrieveError(key, rErr)
			}
		}()
		v, ok := o.results.Load(key)
		if !ok {
			v = f()
			o.results.Store(key, v)
		}
		return v, nil
	})
	if err != nil {
		return err
	}
	return val
}

// Store will store the results for a key in a cache
// This function is not safe to use if multiple subroutines store the
// result for the same key.
func (o *SyncStore) Store(key string, r interface{}) {
	o.results.Store(key, r)
}

// NewSyncStore returns a new instance of `SyncStore`
func NewSyncStore() *SyncStore {
	return &SyncStore{
		sf:      singleflight.Group{},
		results: sync.Map{},
	}
}

// StoreError represent any error that when retrieving errors from the store.
type StoreError struct {
	message string
}

func (e StoreError) Error() string {
	return e.message
}

func retrieveError(key string, i interface{}) StoreError {
	return StoreError{
		message: fmt.Sprintf("internal error retrieving cached results for key %s: %v", key, i),
	}
}
