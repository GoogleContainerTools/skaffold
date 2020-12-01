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

package tracker

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"

	"knative.dev/pkg/apis"
)

// Reference is modeled after corev1.ObjectReference, but omits fields
// unsupported by the tracker, and permits us to extend things in
// divergent ways.
type Reference struct {
	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// Kind of the referent.
	// +optional
	Kind string `json:"kind,omitempty"`

	// Namespace of the referent.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the referent.
	// Mutually exclusive with Selector.
	// +optional
	Name string `json:"name,omitempty"`

	// Selector of the referents.
	// Mutually exclusive with Name.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// Interface defines the interface through which an object can register
// that it is tracking another object by reference.
type Interface interface {
	// Track tells us that "obj" is tracking changes to the
	// referenced object.
	// DEPRECATED: use TrackReference
	Track(ref corev1.ObjectReference, obj interface{}) error

	// Track tells us that "obj" is tracking changes to the
	// referenced object.
	TrackReference(ref Reference, obj interface{}) error

	// OnChanged is a callback to register with the InformerFactory
	// so that we are notified for appropriate object changes.
	OnChanged(obj interface{})

	// GetObservers returns the names of all observers for the given
	// object.
	GetObservers(obj interface{}) []types.NamespacedName

	// OnDeletedObserver is a callback to register with the InformerFactory
	// so that we are notified for deletions of a watching parent to
	// remove the respective tracking.
	OnDeletedObserver(obj interface{})
}

// GroupVersionKind returns the GroupVersion of the object referenced.
func (ref *Reference) GroupVersionKind() schema.GroupVersionKind {
	gv, _ := schema.ParseGroupVersion(ref.APIVersion)
	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    ref.Kind,
	}
}

// ObjectReference returns the tracker Reference as an ObjectReference.
func (ref *Reference) ObjectReference() corev1.ObjectReference {
	return corev1.ObjectReference{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Namespace:  ref.Namespace,
		Name:       ref.Name,
	}
}

// ValidateObjectReference validates that the Reference uses a subset suitable for
// translation to a corev1.ObjectReference.  This helper is intended to simplify
// validating a particular (narrow) use of tracker.Reference.
func (ref *Reference) ValidateObjectReference(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	// Required fields
	if ref.APIVersion == "" {
		errs = errs.Also(apis.ErrMissingField("apiVersion"))
	} else if verrs := validation.IsQualifiedName(ref.APIVersion); len(verrs) != 0 {
		errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "apiVersion"))
	}
	if ref.Kind == "" {
		errs = errs.Also(apis.ErrMissingField("kind"))
	} else if verrs := validation.IsCIdentifier(ref.Kind); len(verrs) != 0 {
		errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "kind"))
	}
	if ref.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	} else if verrs := validation.IsDNS1123Label(ref.Name); len(verrs) != 0 {
		errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "name"))
	}
	if ref.Namespace == "" {
		errs = errs.Also(apis.ErrMissingField("namespace"))
	} else if verrs := validation.IsDNS1123Label(ref.Namespace); len(verrs) != 0 {
		errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "namespace"))
	}

	// Disallowed fields in ObjectReference-compatible context.
	if ref.Selector != nil {
		errs = errs.Also(apis.ErrDisallowedFields("selector"))
	}

	return errs
}

func (ref *Reference) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	// Required fields
	if ref.APIVersion == "" {
		errs = errs.Also(apis.ErrMissingField("apiVersion"))
	} else if verrs := validation.IsQualifiedName(ref.APIVersion); len(verrs) != 0 {
		errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "apiVersion"))
	}
	if ref.Kind == "" {
		errs = errs.Also(apis.ErrMissingField("kind"))
	} else if verrs := validation.IsCIdentifier(ref.Kind); len(verrs) != 0 {
		errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "kind"))
	}
	if ref.Namespace == "" {
		errs = errs.Also(apis.ErrMissingField("namespace"))
	} else if verrs := validation.IsDNS1123Label(ref.Namespace); len(verrs) != 0 {
		errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "namespace"))
	}

	switch {
	case ref.Selector != nil && ref.Name != "":
		errs = errs.Also(apis.ErrMultipleOneOf("selector", "name"))
	case ref.Selector != nil:
		_, err := metav1.LabelSelectorAsSelector(ref.Selector)
		if err != nil {
			errs = errs.Also(apis.ErrInvalidValue(err.Error(), "selector"))
		}

	case ref.Name != "":
		if verrs := validation.IsDNS1123Label(ref.Name); len(verrs) != 0 {
			errs = errs.Also(apis.ErrInvalidValue(strings.Join(verrs, ", "), "name"))
		}
	default:
		errs = errs.Also(apis.ErrMissingOneOf("selector", "name"))
	}

	return errs

}
