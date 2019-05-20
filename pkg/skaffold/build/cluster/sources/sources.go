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

package sources

import (
	"context"
	"io"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildContextSource is the generic type for the different build context sources the kaniko builder can use
type BuildContextSource interface {
	Setup(ctx context.Context, out io.Writer, artifact *latest.Artifact, initialTag string, dependencies []string) (string, error)
	Pod(args []string) *v1.Pod
	ModifyPod(ctx context.Context, p *v1.Pod) error
	Cleanup(ctx context.Context) error
}

// Retrieve returns the correct build context based on the config
func Retrieve(clusterDetails *latest.ClusterDetails, artifact *latest.KanikoArtifact) BuildContextSource {
	if artifact.BuildContext.LocalDir != nil {
		return &LocalDir{
			clusterDetails: clusterDetails,
			artifact:       artifact,
		}
	}

	return &GCSBucket{
		clusterDetails: clusterDetails,
		artifact:       artifact,
	}
}

func podTemplate(clusterDetails *latest.ClusterDetails, artifact *latest.KanikoArtifact, args []string) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko-",
			Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			Namespace:    clusterDetails.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            constants.DefaultKanikoContainerName,
					Image:           artifact.Image,
					Args:            args,
					ImagePullPolicy: v1.PullIfNotPresent,
					Env: []v1.EnvVar{{
						Name:  "GOOGLE_APPLICATION_CREDENTIALS",
						Value: "/secret/kaniko-secret",
					}},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      constants.DefaultKanikoSecretName,
							MountPath: "/secret",
						},
					},
					Resources: resourceRequirements(clusterDetails.Resources),
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{{
				Name: constants.DefaultKanikoSecretName,
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: clusterDetails.PullSecretName,
					},
				},
			},
			},
		},
	}

	if artifact.Cache != nil && artifact.Cache.HostPath != "" {
		volumeMount := v1.VolumeMount{
			Name:      constants.DefaultKanikoCacheDirName,
			MountPath: constants.DefaultKanikoCacheDirMountPath,
		}

		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, volumeMount)

		volume := v1.Volume{
			Name: constants.DefaultKanikoCacheDirName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: artifact.Cache.HostPath,
				},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
	}

	if clusterDetails.DockerConfig == nil {
		return pod
	}

	volumeMount := v1.VolumeMount{
		Name:      constants.DefaultKanikoDockerConfigSecretName,
		MountPath: constants.DefaultKanikoDockerConfigPath,
	}

	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, volumeMount)

	volume := v1.Volume{
		Name: constants.DefaultKanikoDockerConfigSecretName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: clusterDetails.DockerConfig.SecretName,
			},
		},
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

	return pod
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
		}

		if rr.Requests != nil {
			req.Requests = v1.ResourceList{}
			if rr.Requests.CPU != "" {
				req.Requests[v1.ResourceCPU] = resource.MustParse(rr.Requests.CPU)
			}
			if rr.Requests.Memory != "" {
				req.Requests[v1.ResourceMemory] = resource.MustParse(rr.Requests.Memory)
			}
		}
	}

	return req

}
