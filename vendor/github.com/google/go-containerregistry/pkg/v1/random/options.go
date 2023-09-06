// Copyright 2018 Google LLC All Rights Reserved.
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

package random

import "math/rand"

// Option is an optional parameter to the random functions
type Option func(opts *options)

type options struct {
	source rand.Source

	// TODO opens the door to add this in the future
	// algorithm digest.Algorithm
}

func getOptions(opts []Option) *options {
	// get a random seed

	// TODO in go 1.20 this is fine (it will be random)
	seed := rand.Int63() //nolint:gosec
	/*
		// in prior go versions this needs to come from crypto/rand
		var b [8]byte
		_, err := crypto_rand.Read(b[:])
		if err != nil {
			panic("cryptographically secure random number generator is not working")
		}
		seed := int64(binary.LittleEndian.Int64(b[:]))
	*/

	// defaults
	o := &options{
		source: rand.NewSource(seed),
	}

	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithSource sets the random number generator source
func WithSource(source rand.Source) Option {
	return func(opts *options) {
		opts.source = source
	}
}
