/*
Copyright 2019 The Tekton Authors.

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
	"fmt"
	"net/url"

	"github.com/knative/pkg/apis"
)

// AllResultTargetTypes is a list of all ResultTargetTypes, used for validation
var AllResultTargetTypes = []ResultTargetType{ResultTargetTypeGCS}

// ResultTargetType represents the type of endpoint that this result target is,
// so that the controller will know how to write results to it.
type ResultTargetType string

const (
	// ResultTargetTypeGCS indicates that the URL endpoint is a GCS bucket.
	ResultTargetTypeGCS = "gcs"
)

// Results is used to identify an endpoint where results can be uploaded. The
// serviceaccount used for the pipeline must have access to this endpoint.
type Results struct {
	Type ResultTargetType `json:"type"`
	URL  string           `json:"url"`
}

// Validate will validate the result configuration. The path is the path at which
// we found this instance of `Results` (since it is probably a member of another
// structure) and will be used to report any errors.
func (r *Results) Validate(ctx context.Context, path string) *apis.FieldError {
	if r.Type != ResultTargetTypeGCS {
		return apis.ErrInvalidValue(string(r.Type), fmt.Sprintf("%s.Type", path))
	}

	if err := validateResultTargetType(r.Type, fmt.Sprintf("%s.Type", path)); err != nil {
		return err
	}

	if r.URL == "" {
		return apis.ErrMissingField(fmt.Sprintf("%s.URL", path))
	}

	if err := validateURL(r.URL, fmt.Sprintf("%s.URL", path)); err != nil {
		return err
	}
	return nil
}

func validateURL(u, path string) *apis.FieldError {
	if u == "" {
		return nil
	}
	_, err := url.ParseRequestURI(u)
	if err != nil {
		return apis.ErrInvalidValue(u, path)
	}
	return nil
}

func validateResultTargetType(r ResultTargetType, path string) *apis.FieldError {
	for _, a := range AllResultTargetTypes {
		if a == r {
			return nil
		}
	}
	return apis.ErrInvalidValue(string(r), path)
}
