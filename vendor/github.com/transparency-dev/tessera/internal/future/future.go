// Copyright 2025 The Tessera authors. All Rights Reserved.
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

package future

import "sync"

// FutureErr is a future which resolves to a value or an error.
type FutureErr[T any] struct {
	// done is used to block/unblock calls to resolve the future.
	//
	// This could be done with a channel, but that turns out to be heavier in terms of
	// memory alloc than using a waitgroup.
	done *sync.WaitGroup
	val  T
	err  error
}

// NewFutureErr creates a new future which resolves to a T or an error.
//
// Returns the future, and a function which is used to set the future's value/error.
// Calls to the future's Get function will block until this function is called.
func NewFutureErr[T any]() (*FutureErr[T], func(T, error)) {
	f := &FutureErr[T]{
		done: &sync.WaitGroup{},
	}
	f.done.Add(1)
	var o sync.Once
	return f, func(t T, err error) {
		o.Do(func() {
			f.val = t
			f.err = err
			f.done.Done()
		})
	}
}

// Get resolves the future, returning either a valid T or an error.
//
// This function will block until the future has had its value set.
func (f *FutureErr[T]) Get() (T, error) {
	f.done.Wait()
	return f.val, f.err
}
