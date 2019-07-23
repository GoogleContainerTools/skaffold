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
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tektoncd/pipeline/pkg/names"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
)

var (
	gsutilImage              = flag.String("gsutil-image", "override-with-gsutil-image:latest", "The container image containing gsutil")
	gcsSecretVolumeMountPath = "/var/secret"
)

// GCSResource is a GCS endpoint from which to get artifacts which is required
// by a Build/Task for context (e.g. a archive from which to build an image).
type GCSResource struct {
	Name           string               `json:"name"`
	Type           PipelineResourceType `json:"type"`
	Location       string               `json:"location"`
	TypeDir        bool                 `json:"typeDir"`
	DestinationDir string               `json:"destinationDir"`
	//Secret holds a struct to indicate a field name and corresponding secret name to populate it
	Secrets []SecretParam `json:"secrets"`
}

// NewGCSResource creates a new GCS resource to pass to a Task
func NewGCSResource(r *PipelineResource) (*GCSResource, error) {
	if r.Spec.Type != PipelineResourceTypeStorage {
		return nil, xerrors.Errorf("GCSResource: Cannot create a GCS resource from a %s Pipeline Resource", r.Spec.Type)
	}
	var location string
	var locationSpecified, dir bool

	for _, param := range r.Spec.Params {
		switch {
		case strings.EqualFold(param.Name, "Location"):
			location = param.Value
			if param.Value != "" {
				locationSpecified = true
			}
		case strings.EqualFold(param.Name, "Dir"):
			dir = true // if dir flag is present then its a dir
		}
	}

	if !locationSpecified {
		return nil, xerrors.Errorf("GCSResource: Need Location to be specified in order to create GCS resource %s", r.Name)
	}
	return &GCSResource{
		Name:     r.Name,
		Type:     r.Spec.Type,
		Location: location,
		TypeDir:  dir,
		Secrets:  r.Spec.SecretParams,
	}, nil
}

// GetName returns the name of the resource
func (s GCSResource) GetName() string {
	return s.Name
}

// GetType returns the type of the resource, in this case "storage"
func (s GCSResource) GetType() PipelineResourceType {
	return PipelineResourceTypeStorage
}

// GetSecretParams returns the resource secret params
func (s *GCSResource) GetSecretParams() []SecretParam { return s.Secrets }

// Replacements is used for template replacement on an GCSResource inside of a Taskrun.
func (s *GCSResource) Replacements() map[string]string {
	return map[string]string{
		"name":     s.Name,
		"type":     string(s.Type),
		"location": s.Location,
		"path":     s.DestinationDir,
	}
}

// SetDestinationDirectory sets the destination directory at runtime like where is the resource going to be copied to
func (s *GCSResource) SetDestinationDirectory(destDir string) { s.DestinationDir = destDir }

// GetUploadContainerSpec gets container spec for gcs resource to be uploaded like
// set environment variable from secret params and set volume mounts for those secrets
func (s *GCSResource) GetUploadContainerSpec() ([]corev1.Container, error) {
	if s.DestinationDir == "" {
		return nil, xerrors.Errorf("GCSResource: Expect Destination Directory param to be set: %s", s.Name)
	}
	var args []string
	if s.TypeDir {
		args = []string{"-args", fmt.Sprintf("rsync -d -r %s %s", s.DestinationDir, s.Location)}
	} else {
		args = []string{"-args", fmt.Sprintf("cp %s %s", filepath.Join(s.DestinationDir, "*"), s.Location)}
	}

	envVars, secretVolumeMount := getSecretEnvVarsAndVolumeMounts(s.Name, gcsSecretVolumeMountPath, s.Secrets)

	return []corev1.Container{{
		Name:         names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("upload-%s", s.Name)),
		Image:        *gsutilImage,
		Command:      []string{"/ko-app/gsutil"},
		Args:         args,
		VolumeMounts: secretVolumeMount,
		Env:          envVars,
	}}, nil
}

// GetDownloadContainerSpec returns an array of container specs to download gcs storage object
func (s *GCSResource) GetDownloadContainerSpec() ([]corev1.Container, error) {
	if s.DestinationDir == "" {
		return nil, xerrors.Errorf("GCSResource: Expect Destination Directory param to be set %s", s.Name)
	}
	var args []string
	if s.TypeDir {
		args = []string{"-args", fmt.Sprintf("rsync -d -r %s %s", s.Location, s.DestinationDir)}
	} else {
		args = []string{"-args", fmt.Sprintf("cp %s %s", s.Location, s.DestinationDir)}
	}

	envVars, secretVolumeMount := getSecretEnvVarsAndVolumeMounts(s.Name, gcsSecretVolumeMountPath, s.Secrets)
	return []corev1.Container{
		CreateDirContainer(s.Name, s.DestinationDir), {
			Name:         names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("fetch-%s", s.Name)),
			Image:        *gsutilImage,
			Command:      []string{"/ko-app/gsutil"},
			Args:         args,
			Env:          envVars,
			VolumeMounts: secretVolumeMount,
		}}, nil
}
