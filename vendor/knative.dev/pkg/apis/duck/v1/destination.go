/*
Copyright 2019 The Knative Authors

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

package v1

import (
	"context"

	"knative.dev/pkg/apis"
)

// Destination represents a target of an invocation over HTTP.
type Destination struct {
	// Ref points to an Addressable.
	// +optional
	Ref *KReference `json:"ref,omitempty"`

	// URI can be an absolute URL(non-empty scheme and non-empty host) pointing to the target or a relative URI. Relative URIs will be resolved using the base URI retrieved from Ref.
	// +optional
	URI *apis.URL `json:"uri,omitempty"`
}

// Validate the Destination has all the necessary fields and check the
// Namespace matches that of the parent object (using apis.ParentMeta).
func (dest *Destination) Validate(ctx context.Context) *apis.FieldError {
	if dest == nil {
		return nil
	}
	return ValidateDestination(ctx, *dest).ViaField(apis.CurrentField)
}

// ValidateDestination validates Destination.
func ValidateDestination(ctx context.Context, dest Destination) *apis.FieldError {
	ref := dest.Ref
	uri := dest.URI
	if ref == nil && uri == nil {
		return apis.ErrGeneric("expected at least one, got none", "ref", "uri")
	}

	if ref != nil && uri != nil && uri.URL().IsAbs() {
		return apis.ErrGeneric("Absolute URI is not allowed when Ref or [apiVersion, kind, name] is present", "[apiVersion, kind, name]", "ref", "uri")
	}
	// IsAbs() check whether the URL has a non-empty scheme. Besides the non-empty scheme, we also require uri has a non-empty host
	if ref == nil && uri != nil && (!uri.URL().IsAbs() || uri.Host == "") {
		return apis.ErrInvalidValue("Relative URI is not allowed when Ref and [apiVersion, kind, name] is absent", "uri")
	}
	if ref != nil && uri == nil {
		return ref.Validate(ctx).ViaField("ref")
	}
	return nil
}

// GetRef gets the KReference from this Destination, if one is present. If no ref is present,
// then nil is returned.
func (dest *Destination) GetRef() *KReference {
	if dest == nil {
		return nil
	}
	return dest.Ref
}

func (d *Destination) SetDefaults(ctx context.Context) {
	if d.Ref != nil && d.Ref.Namespace == "" {
		d.Ref.Namespace = apis.ParentMeta(ctx).Namespace
	}
}
