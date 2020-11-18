/*
Copyright 2020 The Knative Authors

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

package ducktypes

import (
	"knative.dev/pkg/apis"
)

// Implementable is implemented by the Fooable duck type that consumers
// are expected to embed as a `.status.fooable` field.
type Implementable interface {
	// GetFullType returns an instance of a full resource wrapping
	// an instance of this Implementable that can populate its fields
	// to verify json roundtripping.
	GetFullType() Populatable
}

// Populatable is implemented by a skeleton resource wrapping an Implementable
// duck type.  It will generally have TypeMeta, ObjectMeta, and a Status field
// wrapping a Fooable field.
type Populatable interface {
	apis.Listable

	// Populate fills in all possible fields, so that we can verify that
	// they roundtrip properly through JSON.
	Populate()
}

const GroupName = "duck.knative.dev"
