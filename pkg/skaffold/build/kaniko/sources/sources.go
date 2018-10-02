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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildContextSource is the generic type for the different build context sources the kaniko builder can use
type BuildContextSource interface {
	Setup(ctx context.Context, artifact *latest.Artifact, cfg *latest.KanikoBuild, initialTag string) (string, error)
	Pod(cfg *latest.KanikoBuild, args []string) *v1.Pod
	ModifyPod(p *v1.Pod) error
	Cleanup(ctx context.Context, cfg *latest.KanikoBuild) error
}

// Retrieve returns the correct build context based on the config
func Retrieve(cfg *latest.KanikoBuild) (BuildContextSource, error) {
	if cfg.BuildContext.GCSBucket != "" {
		return &GCSBucket{}, nil
	}
	if cfg.BuildContext.LocalDir != nil {
		return &LocalDir{}, nil
	}
	return nil, errors.New("no valid build context was provided for kaniko builder")
}

func podTemplate(cfg *latest.KanikoBuild, args []string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko",
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
			}},
		},
	}
}
