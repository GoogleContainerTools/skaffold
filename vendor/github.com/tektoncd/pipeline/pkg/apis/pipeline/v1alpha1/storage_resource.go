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
	"strings"

	"golang.org/x/xerrors"
)

type PipelineResourceStorageType string

const (
	// PipelineResourceTypeGCS indicates that resource source is a GCS blob/directory.
	PipelineResourceTypeGCS      PipelineResourceType = "gcs"
	PipelineResourceTypeBuildGCS PipelineResourceType = "build-gcs"
)

// PipelineResourceInterface interface to be implemented by different PipelineResource types
type PipelineStorageResourceInterface interface {
	PipelineResourceInterface
	GetSecretParams() []SecretParam
}

func NewStorageResource(r *PipelineResource) (PipelineStorageResourceInterface, error) {
	if r.Spec.Type != PipelineResourceTypeStorage {
		return nil, xerrors.Errorf("StoreResource: Cannot create a storage resource from a %s Pipeline Resource", r.Spec.Type)
	}

	for _, param := range r.Spec.Params {
		if strings.EqualFold(param.Name, "type") {
			switch {
			case strings.EqualFold(param.Value, string(PipelineResourceTypeGCS)):
				return NewGCSResource(r)
			case strings.EqualFold(param.Value, string(PipelineResourceTypeBuildGCS)):
				return NewBuildGCSResource(r)
			default:
				return nil, xerrors.Errorf("%s is an invalid or unimplemented PipelineStorageResource", param.Value)
			}
		}
	}
	return nil, xerrors.Errorf("StoreResource: Cannot create a storage resource without type %s in spec", r.Name)
}
