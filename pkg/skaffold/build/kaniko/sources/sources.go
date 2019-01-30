/*
Copyright 2018 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildContextSource is the generic type for the different build context sources the kaniko builder can use
type BuildContextSource interface {
	Setup(ctx context.Context, out io.Writer, artifact *latest.Artifact, initialTag string) (string, error)
	Pod(args []string) *v1.Pod
	ModifyPod(ctx context.Context, p *v1.Pod) error
	Cleanup(ctx context.Context) error
}

// Retrieve returns the correct build context based on the config
func Retrieve(cfg *latest.KanikoBuild) BuildContextSource {
	if cfg.BuildContext.LocalDir != nil {
		return &LocalDir{
			cfg: cfg,
		}
	}

	return &GCSBucket{
		cfg: cfg,
	}
}

func podTemplate(cfg *latest.KanikoBuild, args []string) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko-",
			Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			Namespace:    cfg.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            constants.DefaultKanikoContainerName,
					Image:           constants.DefaultKanikoImage,
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
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{{
				Name: constants.DefaultKanikoSecretName,
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: cfg.PullSecretName,
					},
				},
			},
			},
		},
	}

	if cfg.DockerConfig == nil {
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
				SecretName: cfg.DockerConfig.SecretName,
			},
		},
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

	return pod
}
