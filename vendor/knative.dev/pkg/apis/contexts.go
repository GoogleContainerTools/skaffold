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

package apis

import (
	"context"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This is attached to contexts passed to webhook interfaces when
// the receiver being validated is being created.
type inCreateKey struct{}

// WithinCreate is used to note that the webhook is calling within
// the context of a Create operation.
func WithinCreate(ctx context.Context) context.Context {
	return context.WithValue(ctx, inCreateKey{}, struct{}{})
}

// IsInCreate checks whether the context is a Create.
func IsInCreate(ctx context.Context) bool {
	return ctx.Value(inCreateKey{}) != nil
}

// This is attached to contexts passed to webhook interfaces when
// the receiver being validated is being updated.
type inUpdateKey struct{}

type updatePayload struct {
	base        interface{}
	subresource string
}

// WithinUpdate is used to note that the webhook is calling within
// the context of a Update operation.
func WithinUpdate(ctx context.Context, base interface{}) context.Context {
	return context.WithValue(ctx, inUpdateKey{}, &updatePayload{
		base: base,
	})
}

// WithinSubResourceUpdate is used to note that the webhook is calling within
// the context of a Update operation on a subresource.
func WithinSubResourceUpdate(ctx context.Context, base interface{}, sr string) context.Context {
	return context.WithValue(ctx, inUpdateKey{}, &updatePayload{
		base:        base,
		subresource: sr,
	})
}

// IsInUpdate checks whether the context is an Update.
func IsInUpdate(ctx context.Context) bool {
	return ctx.Value(inUpdateKey{}) != nil
}

// IsInStatusUpdate checks whether the context is an Update.
func IsInStatusUpdate(ctx context.Context) bool {
	value := ctx.Value(inUpdateKey{})
	if value == nil {
		return false
	}
	up := value.(*updatePayload)
	return up.subresource == "status"
}

// GetBaseline returns the baseline of the update, or nil when we
// are not within an update context.
func GetBaseline(ctx context.Context) interface{} {
	value := ctx.Value(inUpdateKey{})
	if value == nil {
		return nil
	}
	return value.(*updatePayload).base
}

// This is attached to contexts passed to webhook interfaces when
// the receiver being validated is being created.
type userInfoKey struct{}

// WithUserInfo is used to note that the webhook is calling within
// the context of a Create operation.
func WithUserInfo(ctx context.Context, ui *authenticationv1.UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey{}, ui)
}

// GetUserInfo accesses the UserInfo attached to the webhook context.
func GetUserInfo(ctx context.Context) *authenticationv1.UserInfo {
	if ui, ok := ctx.Value(userInfoKey{}).(*authenticationv1.UserInfo); ok {
		return ui
	}
	return nil
}

// This is attached to contexts as they are passed down through a resource
// being validated or defaulted to signal the ObjectMeta of the enclosing
// resource.
type parentMetaKey struct{}

// WithinParent attaches the ObjectMeta of the resource enclosing the
// nested resources we are validating.  This is intended for use with
// interfaces like apis.Defaultable and apis.Validatable.
func WithinParent(ctx context.Context, om metav1.ObjectMeta) context.Context {
	return context.WithValue(ctx, parentMetaKey{}, om)
}

// ParentMeta accesses the ObjectMeta of the enclosing parent resource
// from the context.  See WithinParent for how to attach the parent's
// ObjectMeta to the context.
func ParentMeta(ctx context.Context) metav1.ObjectMeta {
	if om, ok := ctx.Value(parentMetaKey{}).(metav1.ObjectMeta); ok {
		return om
	}
	return metav1.ObjectMeta{}
}

// This is attached to contexts as they are passed down through a resource
// being validated or defaulted to signal that we are within a Spec.
type inSpec struct{}

// WithinSpec notes on the context that further validation or defaulting
// is within the context of a Spec.  This is intended for use with
// interfaces like apis.Defaultable and apis.Validatable.
func WithinSpec(ctx context.Context) context.Context {
	return context.WithValue(ctx, inSpec{}, struct{}{})
}

// IsInSpec returns whether the context of validation or defaulting is
// the Spec of the parent resource.
func IsInSpec(ctx context.Context) bool {
	return ctx.Value(inSpec{}) != nil
}

// This is attached to contexts as they are passed down through a resource
// being validated or defaulted to signal that we are within a Status.
type inStatus struct{}

// WithinStatus notes on the context that further validation or defaulting
// is within the context of a Status.  This is intended for use with
// interfaces like apis.Defaultable and apis.Validatable.
func WithinStatus(ctx context.Context) context.Context {
	return context.WithValue(ctx, inStatus{}, struct{}{})
}

// IsInStatus returns whether the context of validation or defaulting is
// the Status of the parent resource.
func IsInStatus(ctx context.Context) bool {
	return ctx.Value(inStatus{}) != nil
}

// This is attached to contexts as they are passed down through a resource
// being validated to direct them to disallow deprecated fields.
type disallowDeprecated struct{}

// DisallowDeprecated notes on the context that further validation
// should disallow the used of deprecated fields. This may be used
// to ensure that new paths through resources to a common type don't
// allow the mistakes of old versions to be introduced.
func DisallowDeprecated(ctx context.Context) context.Context {
	return context.WithValue(ctx, disallowDeprecated{}, struct{}{})
}

// IsDeprecatedAllowed checks the context to see whether deprecated fields
// are allowed.
func IsDeprecatedAllowed(ctx context.Context) bool {
	return ctx.Value(disallowDeprecated{}) == nil
}
