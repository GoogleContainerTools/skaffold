/*
Copyright 2019 The Skaffold Authors

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

package cluster

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
)

const (
	// kubernetes.io/arch and kubernetes.io/os are known node labels. See https://kubernetes.io/docs/reference/labels-annotations-taints/
	nodeOperatingSystemLabel = "kubernetes.io/os"
	nodeArchitectureLabel    = "kubernetes.io/arch"
)

func (b *Builder) kanikoPodSpec(artifact *latest.KanikoArtifact, tag string, platforms platform.Matcher) (*v1.Pod, error) {
	args, err := kanikoArgs(artifact, tag, b.cfg.GetInsecureRegistries())
	if err != nil {
		return nil, fmt.Errorf("building args list: %w", err)
	}

	vm := v1.VolumeMount{
		Name:      kaniko.DefaultEmptyDirName,
		MountPath: kaniko.DefaultEmptyDirMountPath,
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:  b.ClusterDetails.Annotations,
			GenerateName: "kaniko-",
			Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			Namespace:    b.ClusterDetails.Namespace,
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{{
				Name:            initContainer,
				Image:           artifact.InitImage,
				ImagePullPolicy: v1.PullIfNotPresent,
				Command:         []string{"sh", "-c", "while [ ! -f /tmp/complete ]; do sleep 1; done"},
				VolumeMounts:    []v1.VolumeMount{vm},
				Resources:       resourceRequirements(b.ClusterDetails.Resources),
			}},
			Containers: []v1.Container{{
				Name:                   kaniko.DefaultContainerName,
				Image:                  artifact.Image,
				ImagePullPolicy:        v1.PullIfNotPresent,
				Args:                   args,
				Env:                    b.env(artifact, b.ClusterDetails.HTTPProxy, b.ClusterDetails.HTTPSProxy),
				VolumeMounts:           []v1.VolumeMount{vm},
				Resources:              resourceRequirements(b.ClusterDetails.Resources),
				TerminationMessagePath: artifact.DigestFile, // setting this lets us get the built image digest from container logs directly
			}},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{{
				Name: vm.Name,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			}},
		},
	}

	// Add secret for pull secret
	if b.ClusterDetails.PullSecretName != "" {
		addSecretVolume(pod, kaniko.DefaultSecretName, b.ClusterDetails.PullSecretMountPath, b.ClusterDetails.PullSecretName)
	}

	// Add host path volume for cache
	if artifact.Cache != nil && artifact.Cache.HostPath != "" {
		addHostPathVolume(pod, kaniko.DefaultCacheDirName, kaniko.DefaultCacheDirMountPath, artifact.Cache.HostPath)
	}

	if b.ClusterDetails.DockerConfig != nil {
		// Add secret for docker config if specified
		addSecretVolume(pod, kaniko.DefaultDockerConfigSecretName, kaniko.DefaultDockerConfigPath, b.ClusterDetails.DockerConfig.SecretName)
	}

	// Add Service Account
	if b.ClusterDetails.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = b.ClusterDetails.ServiceAccountName
	}

	// Add SecurityContext for runAsUser
	if b.ClusterDetails.RunAsUser != nil {
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = &v1.PodSecurityContext{}
		}
		pod.Spec.SecurityContext.RunAsUser = b.ClusterDetails.RunAsUser
	}

	// Add Tolerations for kaniko pod setup
	if len(b.ClusterDetails.Tolerations) > 0 {
		pod.Spec.Tolerations = b.ClusterDetails.Tolerations
	}

	// Add nodeSelector for kaniko pod setup
	if b.ClusterDetails.NodeSelector != nil {
		pod.Spec.NodeSelector = b.ClusterDetails.NodeSelector
	}

	// Add nodeSelector for image target platform.
	// Kaniko doesn't support building cross platform images, so the pod platform needs to match the image target platform.
	if len(platforms.Platforms) == 1 {
		if pod.Spec.NodeSelector == nil {
			pod.Spec.NodeSelector = make(map[string]string)
		}
		if _, found := pod.Spec.NodeSelector[nodeArchitectureLabel]; !found {
			pod.Spec.NodeSelector[nodeArchitectureLabel] = platforms.Platforms[0].Architecture
		}
		if _, found := pod.Spec.NodeSelector[nodeOperatingSystemLabel]; !found {
			pod.Spec.NodeSelector[nodeOperatingSystemLabel] = platforms.Platforms[0].OS
		}
	}

	// Add used-defines Volumes
	pod.Spec.Volumes = append(pod.Spec.Volumes, b.Volumes...)

	// Add user-defined VolumeMounts
	for _, vm := range artifact.VolumeMounts {
		pod.Spec.InitContainers[0].VolumeMounts = append(pod.Spec.InitContainers[0].VolumeMounts, vm)
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, vm)
	}

	return pod, nil
}

func (b *Builder) env(artifact *latest.KanikoArtifact, httpProxy, httpsProxy string) []v1.EnvVar {
	env := []v1.EnvVar{{
		// This should be same https://github.com/GoogleContainerTools/kaniko/blob/77cfb912f3483c204bfd09e1ada44fd200b15a78/pkg/executor/push.go#L49
		Name:  "UPSTREAM_CLIENT_TYPE",
		Value: fmt.Sprintf("UpstreamClient(skaffold-%s)", version.Get().Version),
	}}

	for _, v := range artifact.Env {
		if v.Name != "" && v.Value != "" {
			env = append(env, v)
		}
	}

	if httpProxy != "" {
		env = append(env, v1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: httpProxy,
		})
	}

	if httpsProxy != "" {
		env = append(env, v1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: httpsProxy,
		})
	}

	// if cluster.PullSecretName  is non-empty populate secret path and use as GOOGLE_APPLICATION_CREDENTIALS
	// by default it is not empty, so need to
	if b.ClusterDetails.PullSecretName != "" {
		pullSecretPath := strings.Join(
			[]string{b.ClusterDetails.PullSecretMountPath, b.ClusterDetails.PullSecretPath},
			"/", // linux filepath separator.
		)
		env = append(env, v1.EnvVar{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: pullSecretPath,
		})
	}
	return env
}

func addSecretVolume(pod *v1.Pod, name, mountPath, secretName string) {
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	})

	pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	})
}

func addHostPathVolume(pod *v1.Pod, name, mountPath, path string) {
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	})

	pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: path,
			},
		},
	})
}

func resourceRequirements(rr *latest.ResourceRequirements) v1.ResourceRequirements {
	req := v1.ResourceRequirements{}

	if rr != nil {
		if rr.Limits != nil {
			req.Limits = v1.ResourceList{}
			if rr.Limits.CPU != "" {
				req.Limits[v1.ResourceCPU] = resource.MustParse(rr.Limits.CPU)
			}

			if rr.Limits.Memory != "" {
				req.Limits[v1.ResourceMemory] = resource.MustParse(rr.Limits.Memory)
			}

			if rr.Limits.ResourceStorage != "" {
				req.Limits[v1.ResourceStorage] = resource.MustParse(rr.Limits.ResourceStorage)
			}

			if rr.Limits.EphemeralStorage != "" {
				req.Limits[v1.ResourceEphemeralStorage] = resource.MustParse(rr.Limits.EphemeralStorage)
			}
		}

		if rr.Requests != nil {
			req.Requests = v1.ResourceList{}
			if rr.Requests.CPU != "" {
				req.Requests[v1.ResourceCPU] = resource.MustParse(rr.Requests.CPU)
			}
			if rr.Requests.Memory != "" {
				req.Requests[v1.ResourceMemory] = resource.MustParse(rr.Requests.Memory)
			}
			if rr.Requests.ResourceStorage != "" {
				req.Requests[v1.ResourceStorage] = resource.MustParse(rr.Requests.ResourceStorage)
			}
			if rr.Requests.EphemeralStorage != "" {
				req.Requests[v1.ResourceEphemeralStorage] = resource.MustParse(rr.Requests.EphemeralStorage)
			}
		}
	}

	return req
}

func kanikoArgs(artifact *latest.KanikoArtifact, tag string, insecureRegistries map[string]bool) ([]string, error) {
	for reg := range insecureRegistries {
		artifact.InsecureRegistry = append(artifact.InsecureRegistry, reg)
	}

	// Create pod spec
	args, err := kaniko.Args(artifact, tag, fmt.Sprintf("dir://%s", kaniko.DefaultEmptyDirMountPath))
	if err != nil {
		return nil, fmt.Errorf("unable build kaniko args: %w", err)
	}

	log.Entry(context.TODO()).Trace("kaniko arguments are ", strings.Join(args, " "))

	return args, nil
}
