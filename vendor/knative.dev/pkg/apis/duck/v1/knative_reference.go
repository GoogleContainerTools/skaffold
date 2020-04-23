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

package v1

import (
	"context"
	"fmt"

	"knative.dev/pkg/apis"
)

// KReference contains enough information to refer to another object.
// It's a trimmed down version of corev1.ObjectReference.
type KReference struct {
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	Kind string `json:"kind"`

	// Namespace of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
	// This is optional field, it gets defaulted to the object holding it if left out.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name"`

	// API version of the referent.
	APIVersion string `json:"apiVersion"`
}

func (kr *KReference) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError
	if kr == nil {
		return errs.Also(apis.ErrMissingField("name")).
			Also(apis.ErrMissingField("apiVersion")).
			Also(apis.ErrMissingField("kind"))
	}
	if kr.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	}
	if kr.APIVersion == "" {
		errs = errs.Also(apis.ErrMissingField("apiVersion"))
	}
	if kr.Kind == "" {
		errs = errs.Also(apis.ErrMissingField("kind"))
	}
	// Only if namespace is empty validate it. This is to deal with legacy
	// objects in the storage that may now have the namespace filled in.
	// Because things get defaulted in other cases, moving forward the
	// kr.Namespace will not be empty.
	if kr.Namespace != "" {
		if !apis.IsDifferentNamespaceAllowed(ctx) {
			parentNS := apis.ParentMeta(ctx).Namespace
			if parentNS != "" && kr.Namespace != parentNS {
				errs = errs.Also(&apis.FieldError{
					Message: "mismatched namespaces",
					Paths:   []string{"namespace"},
					Details: fmt.Sprintf("parent namespace: %q does not match ref: %q", parentNS, kr.Namespace),
				})
			}

		}
	}
	return errs
}

// SetDefaults sets the default values on the KReference.
func (kr *KReference) SetDefaults(ctx context.Context) {
	if kr.Namespace == "" {
		kr.Namespace = apis.ParentMeta(ctx).Namespace
	}
}
