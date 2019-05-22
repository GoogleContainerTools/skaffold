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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPod(t *testing.T) {
	localDir := &LocalDir{
		artifact: &latest.KanikoArtifact{
			Image: "image",
			BuildContext: &latest.KanikoBuildContext{
				LocalDir: &latest.LocalDir{
					InitImage: "init/image",
				},
			},
		},
		clusterDetails: &latest.ClusterDetails{
			Namespace:      "ns",
			PullSecretName: "secret",
		},
	}

	pod := localDir.Pod([]string{"arg1", "arg2"})

	expectedPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko-",
			Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			Namespace:    "ns",
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{{
				Name:    initContainer,
				Image:   "init/image",
				Command: []string{"sh", "-c", "while [ ! -f /tmp/complete ]; do sleep 1; done"},
				VolumeMounts: []v1.VolumeMount{{
					Name:      constants.DefaultKanikoEmptyDirName,
					MountPath: constants.DefaultKanikoEmptyDirMountPath,
				}},
			}},
			Containers: []v1.Container{{
				Name:            constants.DefaultKanikoContainerName,
				Image:           "image",
				Args:            []string{"arg1", "arg2"},
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
					{
						Name:      constants.DefaultKanikoEmptyDirName,
						MountPath: constants.DefaultKanikoEmptyDirMountPath,
					},
				},
				Resources: v1.ResourceRequirements{},
			}},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: constants.DefaultKanikoSecretName,
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "secret",
						},
					},
				},
				{
					Name: constants.DefaultKanikoEmptyDirName,
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	testutil.CheckDeepEqual(t, expectedPod, pod)
}
