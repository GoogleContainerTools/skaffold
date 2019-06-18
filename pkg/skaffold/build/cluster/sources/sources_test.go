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

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPodTemplate(t *testing.T) {
	tests := []struct {
		description string
		initial     *latest.ClusterDetails
		image       string
		args        []string
		expected    *v1.Pod
	}{
		{
			description: "basic pod",
			initial:     &latest.ClusterDetails{},
			expected: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "kaniko-",
					Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
				},
				Spec: v1.PodSpec{
					RestartPolicy: "Never",
					Containers: []v1.Container{
						{
							Name: "kaniko",
							Env: []v1.EnvVar{
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/secret/kaniko-secret",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name: "kaniko-secret", MountPath: "/secret",
								},
							},
							ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "kaniko-secret",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "",
								},
							},
						},
					},
				},
			},
		},
		{
			description: "with docker config",
			initial: &latest.ClusterDetails{
				DockerConfig: &latest.DockerConfig{
					SecretName: "docker-cfg",
					Path:       "/kaniko/.docker",
				},
			},
			expected: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "kaniko-",
					Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
				},
				Spec: v1.PodSpec{
					RestartPolicy: "Never",
					Containers: []v1.Container{
						{
							Name: "kaniko",
							Env: []v1.EnvVar{
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/secret/kaniko-secret",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name: "kaniko-secret", MountPath: "/secret",
								},
								{
									Name: "docker-cfg", MountPath: "/kaniko/.docker",
								},
							},
							ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "kaniko-secret",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "",
								},
							},
						},
						{
							Name: "docker-cfg",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "docker-cfg",
								},
							},
						},
					},
				},
			},
		},
		{
			description: "with resource constraints",
			initial: &latest.ClusterDetails{
				Resources: &latest.ResourceRequirements{
					Requests: &latest.ResourceRequirement{
						CPU:    "0.5",
						Memory: "1000",
					},
					Limits: &latest.ResourceRequirement{
						CPU:    "1.0",
						Memory: "2000",
					},
				},
			},
			expected: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "kaniko-",
					Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
				},
				Spec: v1.PodSpec{
					RestartPolicy: "Never",
					Containers: []v1.Container{
						{
							Name: "kaniko",
							Env: []v1.EnvVar{
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/secret/kaniko-secret",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name: "kaniko-secret", MountPath: "/secret",
								},
							},
							ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
							Resources: createResourceRequirements(
								resource.MustParse("1.0"),
								resource.MustParse("2000"),
								resource.MustParse("0.5"),
								resource.MustParse("1000")),
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "kaniko-secret",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "",
								},
							},
						},
					},
				},
			},
		},
	}

	opt := cmp.Comparer(func(x, y resource.Quantity) bool {
		return x.String() == y.String()
	})

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := podTemplate(test.initial, &latest.KanikoArtifact{Image: test.image, Cache: &latest.KanikoCache{}}, test.args)

			t.CheckDeepEqual(test.expected, actual, opt)
		})
	}
}

func createResourceRequirements(cpuLimit resource.Quantity, memoryLimit resource.Quantity, cpuRequest resource.Quantity, memoryRequest resource.Quantity) v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceCPU:    cpuLimit,
			v1.ResourceMemory: memoryLimit,
		},
		Requests: v1.ResourceList{
			v1.ResourceCPU:    cpuRequest,
			v1.ResourceMemory: memoryRequest,
		},
	}
}
