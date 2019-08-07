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
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func getSecretEnvVarsAndVolumeMounts(name, mountPath string, secrets []SecretParam) ([]corev1.EnvVar, []corev1.VolumeMount) {
	mountPaths := make(map[string]struct{})
	var (
		envVars           []corev1.EnvVar
		secretVolumeMount []corev1.VolumeMount
		authVar           bool
	)

	for _, secretParam := range secrets {
		if secretParam.FieldName == "GOOGLE_APPLICATION_CREDENTIALS" && !authVar {
			authVar = true
			mountPath := filepath.Join(mountPath, secretParam.SecretName)

			envVars = append(envVars, corev1.EnvVar{
				Name:  strings.ToUpper(secretParam.FieldName),
				Value: filepath.Join(mountPath, secretParam.SecretKey),
			})

			if _, ok := mountPaths[mountPath]; !ok {
				secretVolumeMount = append(secretVolumeMount, corev1.VolumeMount{
					Name:      fmt.Sprintf("volume-%s-%s", name, secretParam.SecretName),
					MountPath: mountPath,
				})
				mountPaths[mountPath] = struct{}{}
			}
		}
	}
	return envVars, secretVolumeMount
}
