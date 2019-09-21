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
	"fmt"
	"io"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// BuildContextSource is the generic type for the different build context sources the kaniko builder can use
type BuildContextSource interface {
	Setup(ctx context.Context, out io.Writer, artifact *latest.Artifact, initialTag string, dependencies []string) (string, error)
	Pod(args []string) *v1.Pod
	ModifyPod(ctx context.Context, p *v1.Pod) error
	Cleanup(ctx context.Context) error
}

// Retrieve returns the correct build context based on the config
func Retrieve(cli *kubectl.CLI, clusterDetails *latest.ClusterDetails, artifact *latest.KanikoArtifact) BuildContextSource {
	if artifact.BuildContext.LocalDir != nil {
		return &LocalDir{
			clusterDetails: clusterDetails,
			artifact:       artifact,
			kubectl:        cli,
		}
	}

	return &GCSBucket{
		clusterDetails: clusterDetails,
		artifact:       artifact,
	}
}

func podTemplate(clusterDetails *latest.ClusterDetails, artifact *latest.KanikoArtifact, args []string, version string) *v1.Pod {
	userAgent := fmt.Sprintf("UpstreamClient(skaffold-%s)", version)

	env := []v1.EnvVar{{
		Name:  "GOOGLE_APPLICATION_CREDENTIALS",
		Value: "/secret/kaniko-secret",
	}, {
		// This should be same https://github.com/GoogleContainerTools/kaniko/blob/77cfb912f3483c204bfd09e1ada44fd200b15a78/pkg/executor/push.go#L49
		Name:  "UPSTREAM_CLIENT_TYPE",
		Value: userAgent,
	}}

	env = setProxy(clusterDetails, env)

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
					Env:             env,
					Resources:       resourceRequirements(clusterDetails.Resources),
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	// Add secret for pull secret
	if clusterDetails.PullSecretName != "" {
		addSecretVolume(pod, constants.DefaultKanikoSecretName, "/secret", clusterDetails.PullSecretName)
	}

	// Add host path volume for cache
	if artifact.Cache != nil && artifact.Cache.HostPath != "" {
		addHostPathVolume(pod, constants.DefaultKanikoCacheDirName, constants.DefaultKanikoCacheDirMountPath, artifact.Cache.HostPath)
	}

	if clusterDetails.DockerConfig == nil {
		return pod
	}

	// Add secret for docker config if specified
	addSecretVolume(pod, constants.DefaultKanikoDockerConfigSecretName, constants.DefaultKanikoDockerConfigPath, clusterDetails.DockerConfig.SecretName)
	return pod
}

func addSecretVolume(pod *v1.Pod, name, mountPath, secretName string) {
	vm := v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}
	v := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, vm)
	pod.Spec.Volumes = append(pod.Spec.Volumes, v)
}

func addHostPathVolume(pod *v1.Pod, name, mountPath, path string) {
	vm := v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}
	v := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: path,
			},
		},
	}
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, vm)
	pod.Spec.Volumes = append(pod.Spec.Volumes, v)
}

func setProxy(clusterDetails *latest.ClusterDetails, env []v1.EnvVar) []v1.EnvVar {
	if clusterDetails.HTTPProxy != "" {
		proxy := v1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: clusterDetails.HTTPProxy,
		}
		env = append(env, proxy)
	}

	if clusterDetails.HTTPSProxy != "" {
		proxy := v1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: clusterDetails.HTTPSProxy,
		}
		env = append(env, proxy)
	}
	return env
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
