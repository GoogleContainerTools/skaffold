// Copyright 2021 Google LLC All Rights Reserved.
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

package validate

// Option is a functional option for validate.
type Option func(*options)

type options struct {
	fast bool
}

func makeOptions(opts ...Option) options {
	opt := options{
		fast: false,
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

// Fast causes validate to skip reading and digesting layer bytes.
func Fast(o *options) {
	o.fast = true
}
