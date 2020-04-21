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

package apis

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

// Defaultable defines an interface for setting the defaults for the
// uninitialized fields of this instance.
type Defaultable interface {
	SetDefaults(context.Context)
}

// Validatable indicates that a particular type may have its fields validated.
type Validatable interface {
	// Validate checks the validity of this types fields.
	Validate(context.Context) *FieldError
}

// Convertible indicates that a particular type supports conversions to/from
// "higher" versions of the same type.
type Convertible interface {
	// ConvertTo converts the receiver into `to`.
	ConvertTo(ctx context.Context, to Convertible) error

	// ConvertFrom converts `from` into the receiver.
	ConvertFrom(ctx context.Context, from Convertible) error
}

// Listable indicates that a particular type can be returned via the returned
// list type by the API server.
type Listable interface {
	runtime.Object

	GetListType() runtime.Object
}

// Annotatable indicates that a particular type applies various annotations.
// DEPRECATED: Use WithUserInfo / GetUserInfo from within SetDefaults instead.
// The webhook functionality for this has been turned down, which is why this
// interface is empty.
type Annotatable interface{}

// HasSpec indicates that a particular type has a specification information
// and that information is retrievable.
type HasSpec interface {
	// GetUntypedSpec returns the spec of the resource.
	GetUntypedSpec() interface{}
}
