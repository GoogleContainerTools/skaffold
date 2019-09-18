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
	"fmt"
	"strings"

	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
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

func getStorageUploadVolumeSpec(s PipelineStorageResourceInterface, spec *TaskSpec) ([]corev1.Volume, error) {
	var storageVol []corev1.Volume
	mountedSecrets := map[string]string{}

	for _, volume := range spec.Volumes {
		mountedSecrets[volume.Name] = ""
	}

	// Map holds list of secrets that are mounted as volumes
	for _, secretParam := range s.GetSecretParams() {
		volName := fmt.Sprintf("volume-%s-%s", s.GetName(), secretParam.SecretName)

		gcsSecretVolume := corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretParam.SecretName,
				},
			},
		}

		if _, ok := mountedSecrets[volName]; !ok {
			storageVol = append(storageVol, gcsSecretVolume)
			mountedSecrets[volName] = ""
		}
	}
	return storageVol, nil
}
