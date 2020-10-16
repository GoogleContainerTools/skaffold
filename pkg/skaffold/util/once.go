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

type Once struct {
	oncePerKey *sync.Map
	results    *sync.Map
}

// Do calls the function f if and only if it's being called the first time for a specific key (evaluated from the key function).
// If it's called multiple times for the same key only the first call will execute and store the result of f.
// All other calls will be blocked until the running instance of f returns and all of them receive the same result.
func (o *Once) Do(key func() interface{}, f func() interface{}) interface{} {
	k := key()
	once, _ := o.oncePerKey.LoadOrStore(k, new(sync.Once))
	once.(*sync.Once).Do(func() {
		res := f()
		o.results.Store(k, res)
	})

	val, _ := o.results.Load(k)
	return val
}

// NewOnce returns a new instance of `Once`
func NewOnce() *Once {
	return &Once{
		oncePerKey: new(sync.Map),
		results:    new(sync.Map),
	}
}
