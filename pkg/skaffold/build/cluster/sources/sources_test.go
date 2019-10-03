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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodTemplate(t *testing.T) {
	tests := []struct {
		description string
		initial     *latest.ClusterDetails
		artifact    *latest.KanikoArtifact
		args        []string
		expected    *v1.Pod
	}{
		{
			description: "basic pod",
			initial:     &latest.ClusterDetails{},
			artifact: &latest.KanikoArtifact{
				Image: "kaniko-latest",
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
							Name:  "kaniko",
							Image: "kaniko-latest",
							Env: []v1.EnvVar{{
								Name:  "GOOGLE_APPLICATION_CREDENTIALS",
								Value: "/secret/kaniko-secret",
							}, {
								Name:  "UPSTREAM_CLIENT_TYPE",
								Value: "UpstreamClient(skaffold-test)",
							}},
							ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
						},
					},
				},
			},
		},
		{
			description: "with docker config",
			initial: &latest.ClusterDetails{
				PullSecretName:      "pull-secret",
				PullSecretMountPath: "/secret",
				DockerConfig: &latest.DockerConfig{
					SecretName: "docker-cfg",
					Path:       "/kaniko/.docker",
				},
			},
			artifact: &latest.KanikoArtifact{
				Image: "kaniko-latest",
			},
			expected: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "kaniko-",
					Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
				},
				Spec: v1.PodSpec{
					RestartPolicy: "Never",
					Containers: []v1.Container{{
						Name:  "kaniko",
						Image: "kaniko-latest",
						Env: []v1.EnvVar{{
							Name:  "GOOGLE_APPLICATION_CREDENTIALS",
							Value: "/secret/kaniko-secret",
						}, {
							Name:  "UPSTREAM_CLIENT_TYPE",
							Value: "UpstreamClient(skaffold-test)",
						}},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "kaniko-secret",
								MountPath: "/secret",
							},
							{
								Name:      "docker-cfg",
								MountPath: "/kaniko/.docker",
							},
						},
						ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
					}},
					Volumes: []v1.Volume{
						{
							Name: "kaniko-secret",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "pull-secret",
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
			artifact: &latest.KanikoArtifact{
				Image: "kaniko-latest",
			},
			expected: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "kaniko-",
					Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
				},
				Spec: v1.PodSpec{
					RestartPolicy: "Never",
					Containers: []v1.Container{{
						Name:  "kaniko",
						Image: "kaniko-latest",
						Env: []v1.EnvVar{{
							Name:  "GOOGLE_APPLICATION_CREDENTIALS",
							Value: "/secret/kaniko-secret",
						}, {
							Name:  "UPSTREAM_CLIENT_TYPE",
							Value: "UpstreamClient(skaffold-test)",
						}},
						ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
						Resources: createResourceRequirements(
							resource.MustParse("1.0"),
							resource.MustParse("2000"),
							resource.MustParse("0.5"),
							resource.MustParse("1000")),
					}},
				},
			},
		},
		{
			description: "with cache",
			initial:     &latest.ClusterDetails{},
			artifact: &latest.KanikoArtifact{
				Image: "kaniko-latest",
				Cache: &latest.KanikoCache{
					HostPath: "/cache-path",
				},
			},
			expected: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "kaniko-",
					Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
				},
				Spec: v1.PodSpec{
					RestartPolicy: "Never",
					Containers: []v1.Container{{
						Name:  "kaniko",
						Image: "kaniko-latest",
						Env: []v1.EnvVar{{
							Name:  "GOOGLE_APPLICATION_CREDENTIALS",
							Value: "/secret/kaniko-secret",
						}, {
							Name:  "UPSTREAM_CLIENT_TYPE",
							Value: "UpstreamClient(skaffold-test)",
						}},
						VolumeMounts: []v1.VolumeMount{{
							Name:      "kaniko-cache",
							MountPath: "/cache",
						}},
						ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
					}},
					Volumes: []v1.Volume{{
						Name: "kaniko-cache",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/cache-path",
							},
						},
					}},
				},
			},
		},
	}

	opt := cmp.Comparer(func(x, y resource.Quantity) bool {
		return x.String() == y.String()
	})

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := podTemplate(test.initial, test.artifact, test.args, "test")

			t.CheckDeepEqual(test.expected, actual, opt)
		})
	}
}

func TestSetProxy(t *testing.T) {
	tests := []struct {
		description    string
		clusterDetails *latest.ClusterDetails
		env            []v1.EnvVar
		expectedArgs   []v1.EnvVar
	}{
		{
			description:    "no http and https proxy",
			clusterDetails: &latest.ClusterDetails{},
			env:            []v1.EnvVar{},
			expectedArgs:   []v1.EnvVar{},
		}, {
			description: "set http proxy",

			clusterDetails: &latest.ClusterDetails{HTTPProxy: "proxy.com"},
			env:            []v1.EnvVar{},
			expectedArgs:   []v1.EnvVar{{Name: "HTTP_PROXY", Value: "proxy.com"}},
		}, {
			description:    "set https proxy",
			clusterDetails: &latest.ClusterDetails{HTTPSProxy: "proxy.com"},
			env:            []v1.EnvVar{},
			expectedArgs:   []v1.EnvVar{{Name: "HTTPS_PROXY", Value: "proxy.com"}},
		}, {
			description:    "set http and https proxy",
			clusterDetails: &latest.ClusterDetails{HTTPProxy: "proxy.com", HTTPSProxy: "proxy.com"},
			env:            []v1.EnvVar{},
			expectedArgs:   []v1.EnvVar{{Name: "HTTP_PROXY", Value: "proxy.com"}, {Name: "HTTPS_PROXY", Value: "proxy.com"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := setProxy(test.clusterDetails, test.env)

			t.CheckDeepEqual(test.expectedArgs, actual)
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
