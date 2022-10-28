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
type SyncStore[T any] struct {
	sf      singleflight.Group
	results syncMap[T]
}

type syncMap[T any] struct {
	sync.Map
}

func (m *syncMap[T]) Load(k any) (v T, err error, ok bool) {
	val, found := m.Map.Load(k)
	if !found {
		return
	}
	ok = true
	switch t := val.(type) {
	case error:
		err = t
	case T:
		v = t
	}
	return
}

func (m *syncMap[T]) Store(k any, v T, err error) {
	if err != nil {
		m.Map.Store(k, err)
	} else {
		m.Map.Store(k, v)
	}
}

// Exec executes the function f if and only if it's being called the first time for a specific key.
// If it's called multiple times for the same key only the first call will execute and store the result of f.
// All other calls will be blocked until the running instance of f returns and all of them receive the same result.
func (o *SyncStore[T]) Exec(key string, f func() (T, error)) (T, error) {
	val, err := o.sf.Do(key, func() (_ interface{}, err error) {
		// trap any runtime error due to synchronization issues.
		defer func() {
			if rErr := recover(); rErr != nil {
				err = retrieveError(key, rErr)
			}
		}()
		v, err, ok := o.results.Load(key)
		if ok {
			return v, err
		}
		v, err = f()
		o.results.Store(key, v, err)
		return v, err
	})
	var defaultT T
	if err != nil {
		return defaultT, err
	}
	switch t := val.(type) {
	case error:
		return defaultT, t
	case T:
		return t, nil
	default:
		return defaultT, err
	}
}

// Store will store the results for a key in a cache
// This function is not safe to use if multiple subroutines store the
// result for the same key.
func (o *SyncStore[T]) Store(key string, r T, err error) {
	o.results.Store(key, r, err)
}

// NewSyncStore returns a new instance of `SyncStore`
func NewSyncStore[T any]() *SyncStore[T] {
	return &SyncStore[T]{
		sf:      singleflight.Group{},
		results: syncMap[T]{Map: sync.Map{}},
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
