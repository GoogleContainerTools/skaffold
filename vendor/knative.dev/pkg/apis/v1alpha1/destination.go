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

package v1alpha1

import (
	"context"
	"net/url"
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	"knative.dev/pkg/apis"
)

// Destination represents a target of an invocation over HTTP.
type Destination struct {
	// ObjectReference points to an Addressable.
	*corev1.ObjectReference `json:",inline"`

	// URI is for direct URI Designations.
	URI *apis.URL `json:"uri,omitempty"`

	// Path is used with the resulting URL from Addressable ObjectReference or URI. Must start
	// with `/`. An empty path should be represented as the nil value, not `` or `/`.  Will be
	// appended to the path of the resulting URL from the Addressable, or URI.
	Path *string `json:"path,omitempty"`
}

// NewDestination constructs a Destination from an object reference as a convenience.
func NewDestination(obj *corev1.ObjectReference, paths ...string) (*Destination, error) {
	dest := &Destination{
		ObjectReference: obj,
	}
	err := dest.AppendPath(paths...)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

// NewDestinationURI constructs a Destination from a URI.
func NewDestinationURI(uri *apis.URL, paths ...string) (*Destination, error) {
	dest := &Destination{
		URI: uri,
	}
	err := dest.AppendPath(paths...)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

// AppendPath iteratively appends paths to the Destination.
// The path will always begin with "/" unless it is empty.
// An empty path ("" or "/") will always resolve to nil.
func (current *Destination) AppendPath(paths ...string) error {
	// Start with empty string or existing path
	var fullpath string
	if current.Path != nil {
		fullpath = *current.Path
	}

	// Intelligently join all the paths provided
	fullpath = path.Join("/", fullpath, path.Join(paths...))

	// Parse the URL to trim garbage
	urlpath, err := apis.ParseURL(fullpath)
	if err != nil {
		return err
	}

	// apis.ParseURL returns nil if our path was empty, then our path
	// should reflect that it is not set.
	if urlpath == nil {
		current.Path = nil
		return nil
	}

	// A path of "/" adds no information, just toss it
	// Note that urlpath.Path == "" is always false here (joined with "/" above).
	if urlpath.Path == "/" {
		current.Path = nil
		return nil
	}

	// Only use the plain path from the URL
	current.Path = &urlpath.Path
	return nil
}

func (current *Destination) Validate(ctx context.Context) *apis.FieldError {
	if current != nil {
		errs := validateDestination(*current).ViaField(apis.CurrentField)
		if current.Path != nil {
			errs = errs.Also(validateDestinationPath(*current.Path).ViaField("path"))
		}
		return errs
	} else {
		return nil
	}
}

func validateDestination(dest Destination) *apis.FieldError {
	if dest.URI != nil {
		if dest.ObjectReference != nil {
			return apis.ErrMultipleOneOf("uri", "[apiVersion, kind, name]")
		}
		if dest.URI.Host == "" || dest.URI.Scheme == "" {
			return apis.ErrInvalidValue(dest.URI.String(), "uri")
		}
	} else if dest.ObjectReference == nil {
		return apis.ErrMissingOneOf("uri", "[apiVersion, kind, name]")
	} else {
		return validateDestinationRef(*dest.ObjectReference)
	}
	return nil
}

func validateDestinationPath(path string) *apis.FieldError {
	if strings.HasPrefix(path, "/") {
		if pu, err := url.Parse(path); err != nil {
			return apis.ErrInvalidValue(path, apis.CurrentField)
		} else if !equality.Semantic.DeepEqual(pu, &url.URL{Path: pu.Path}) {
			return apis.ErrInvalidValue(path, apis.CurrentField)
		}
	} else {
		return apis.ErrInvalidValue(path, apis.CurrentField)
	}
	return nil
}

func validateDestinationRef(ref corev1.ObjectReference) *apis.FieldError {
	// Check the object.
	var errs *apis.FieldError
	// Required Fields
	if ref.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	}
	if ref.APIVersion == "" {
		errs = errs.Also(apis.ErrMissingField("apiVersion"))
	}
	if ref.Kind == "" {
		errs = errs.Also(apis.ErrMissingField("kind"))
	}

	return errs
}
