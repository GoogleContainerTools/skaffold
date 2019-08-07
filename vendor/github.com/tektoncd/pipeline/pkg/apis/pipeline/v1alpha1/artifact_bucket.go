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

	"github.com/tektoncd/pipeline/pkg/names"
	corev1 "k8s.io/api/core/v1"
)

const (
	// BucketConfigName is the name of the configmap containing all
	// customizations for the storage bucket.
	BucketConfigName = "config-artifact-bucket"

	// BucketLocationKey is the name of the configmap entry that specifies
	// loction of the bucket.
	BucketLocationKey = "location"

	// BucketServiceAccountSecretName is the name of the configmap entry that specifies
	// the name of the secret that will provide the servie account with bucket access.
	// This secret must  have a key called serviceaccount that will have a value with
	// the service account with access to the bucket
	BucketServiceAccountSecretName = "bucket.service.account.secret.name"

	// BucketServiceAccountSecretKey is the name of the configmap entry that specifies
	// the secret key that will have a value with the service account json with access
	// to the bucket
	BucketServiceAccountSecretKey = "bucket.service.account.secret.key"
)

const (
	// PipelineResourceTypeGit indicates that this source is a GitHub repo.
	ArtifactStorageBucketType = "bucket"

	// PipelineResourceTypeStorage indicates that this source is a storage blob resource.
	ArtifactStoragePVCType = "pvc"
)

var (
	secretVolumeMountPath = "/var/bucketsecret"
)

// ArtifactBucket contains the Storage bucket configuration defined in the
// Bucket config map.
type ArtifactBucket struct {
	Name     string
	Location string
	Secrets  []SecretParam
}

// GetType returns the type of the artifact storage
func (b *ArtifactBucket) GetType() string {
	return ArtifactStorageBucketType
}

// StorageBasePath returns the path to be used to store artifacts in a pipelinerun temporary storage
func (b *ArtifactBucket) StorageBasePath(pr *PipelineRun) string {
	return fmt.Sprintf("%s-%s-bucket", pr.Name, pr.Namespace)
}

// GetCopyFromStorageToContainerSpec returns a container used to download artifacts from temporary storage
func (b *ArtifactBucket) GetCopyFromStorageToContainerSpec(name, sourcePath, destinationPath string) []corev1.Container {
	args := []string{"-args", fmt.Sprintf("cp -P -r %s %s", fmt.Sprintf("%s/%s/*", b.Location, sourcePath), destinationPath)}

	envVars, secretVolumeMount := getSecretEnvVarsAndVolumeMounts("bucket", secretVolumeMountPath, b.Secrets)

	return []corev1.Container{{
		Name:    names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("artifact-dest-mkdir-%s", name)),
		Image:   *BashNoopImage,
		Command: []string{"/ko-app/bash"},
		Args: []string{
			"-args", strings.Join([]string{"mkdir", "-p", destinationPath}, " "),
		},
	}, {
		Name:         names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("artifact-copy-from-%s", name)),
		Image:        *gsutilImage,
		Command:      []string{"/ko-app/gsutil"},
		Args:         args,
		Env:          envVars,
		VolumeMounts: secretVolumeMount,
	}}
}

// GetCopyToStorageFromContainerSpec returns a container used to upload artifacts for temporary storage
func (b *ArtifactBucket) GetCopyToStorageFromContainerSpec(name, sourcePath, destinationPath string) []corev1.Container {
	args := []string{"-args", fmt.Sprintf("cp -P -r %s %s", sourcePath, fmt.Sprintf("%s/%s", b.Location, destinationPath))}

	envVars, secretVolumeMount := getSecretEnvVarsAndVolumeMounts("bucket", secretVolumeMountPath, b.Secrets)

	return []corev1.Container{{
		Name:         names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("artifact-copy-to-%s", name)),
		Image:        *gsutilImage,
		Command:      []string{"/ko-app/gsutil"},
		Args:         args,
		Env:          envVars,
		VolumeMounts: secretVolumeMount,
	}}
}

// GetSecretsVolumes returns the list of volumes for secrets to be mounted
// on pod
func (b *ArtifactBucket) GetSecretsVolumes() []corev1.Volume {
	volumes := []corev1.Volume{}
	for _, sec := range b.Secrets {
		volumes = append(volumes, corev1.Volume{
			Name: fmt.Sprintf("volume-bucket-%s", sec.SecretName),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: sec.SecretName,
				},
			},
		})
	}
	return volumes
}
