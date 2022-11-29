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
	"testing"

	specs "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestKanikoArgs(t *testing.T) {
	tests := []struct {
		description        string
		artifact           *latest.KanikoArtifact
		insecureRegistries map[string]bool
		tag                string
		shouldErr          bool
		expectedArgs       []string
	}{
		{
			description: "simple build",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
			},
			expectedArgs: []string{},
		},
		{
			description: "cache layers",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache:          &latest.KanikoCache{},
			},
			expectedArgs: []string{kaniko.CacheFlag},
		},
		{
			description: "cache layers to specific repo",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache: &latest.KanikoCache{
					Repo: "repo",
				},
			},
			expectedArgs: []string{"--cache", kaniko.CacheRepoFlag, "repo"},
		},
		{
			description: "cache path",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache: &latest.KanikoCache{
					HostPath: "/cache",
				},
			},
			expectedArgs: []string{
				kaniko.CacheFlag,
				kaniko.CacheDirFlag, "/cache"},
		},
		{
			description: "target",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Target:         "target",
			},
			expectedArgs: []string{kaniko.TargetFlag, "target"},
		},
		{
			description: "reproducible",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Reproducible:   true,
			},
			expectedArgs: []string{kaniko.ReproducibleFlag},
		},
		{
			description: "build args",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs: map[string]*string{
					"nil_key":   nil,
					"empty_key": util.Ptr(""),
					"value_key": util.Ptr("value"),
				},
			},
			expectedArgs: []string{
				kaniko.BuildArgsFlag, "empty_key=",
				kaniko.BuildArgsFlag, "nil_key",
				kaniko.BuildArgsFlag, "value_key=value"},
		},
		{
			description: "invalid build args",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs: map[string]*string{
					"invalid": util.Ptr("{{Invalid"),
				},
			},
			shouldErr: true,
		},
		{
			description: "insecure registries",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
			},
			insecureRegistries: map[string]bool{"localhost:4000": true},
			expectedArgs:       []string{kaniko.InsecureRegistryFlag, "localhost:4000"},
		},
		{
			description: "skip tls",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SkipTLS:        true,
			},
			expectedArgs: []string{
				kaniko.SkipTLSFlag,
				kaniko.SkipTLSVerifyRegistryFlag, "gcr.io",
			},
		},
		{
			description: "invalid registry",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SkipTLS:        true,
			},
			tag:       "!!!!",
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			commonArgs := []string{"--destination", "gcr.io/tag", "--dockerfile", "Dockerfile", "--context", "dir:///kaniko/buildcontext"}

			tag := "gcr.io/tag"
			if test.tag != "" {
				tag = test.tag
			}
			args, err := kanikoArgs(test.artifact, tag, test.insecureRegistries)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(append(commonArgs, test.expectedArgs...), args)
			}
		})
	}
}

func TestKanikoPodSpec(t *testing.T) {
	artifact := &latest.KanikoArtifact{
		Image:          "image",
		DockerfilePath: "Dockerfile",
		InitImage:      "init/image",
		Env: []v1.EnvVar{{
			Name:  "KEY",
			Value: "VALUE",
		}},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "cm-volume-1",
				ReadOnly:  true,
				MountPath: "/cm-test-mount-path",
				SubPath:   "/subpath",
			},
			{
				Name:      "secret-volume-1",
				ReadOnly:  true,
				MountPath: "/secret-test-mount-path",
				SubPath:   "/subpath",
			},
		},
	}

	var runAsUser int64 = 0

	builder := &Builder{
		cfg: &mockBuilderContext{},
		ClusterDetails: &latest.ClusterDetails{
			Namespace:           "ns",
			PullSecretName:      "secret",
			PullSecretPath:      "kaniko-secret.json",
			PullSecretMountPath: "/secret",
			HTTPProxy:           "http://proxy",
			HTTPSProxy:          "https://proxy",
			ServiceAccountName:  "aVerySpecialSA",
			RunAsUser:           &runAsUser,
			Resources: &latest.ResourceRequirements{
				Requests: &latest.ResourceRequirement{
					CPU: "0.1",
				},
				Limits: &latest.ResourceRequirement{
					CPU: "0.5",
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "cm-volume-1",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "cm-1",
							},
						},
					},
				},
				{
					Name: "secret-volume-1",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "secret-1",
						},
					},
				},
			},
			Tolerations: []v1.Toleration{
				{
					Key:               "app",
					Operator:          "Equal",
					Value:             "skaffold",
					Effect:            "NoSchedule",
					TolerationSeconds: nil,
				},
			},
			NodeSelector: map[string]string{"kubernetes.io/os": "linux"},
		},
	}
	matcher := platform.Matcher{Platforms: []specs.Platform{{OS: "linux", Architecture: "arm64"}}}
	pod, _ := builder.kanikoPodSpec(artifact, "tag", matcher)

	expectedPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:  map[string]string{"test": "test"},
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
					Name:      kaniko.DefaultEmptyDirName,
					MountPath: kaniko.DefaultEmptyDirMountPath,
				}, {
					Name:      "cm-volume-1",
					ReadOnly:  true,
					MountPath: "/cm-secret-mount-path",
					SubPath:   "/subpath",
				}, {
					Name:      "secret-volume-1",
					ReadOnly:  true,
					MountPath: "/secret-secret-mount-path",
					SubPath:   "/subpath",
				}},
				Resources: v1.ResourceRequirements{
					Requests: map[v1.ResourceName]resource.Quantity{
						v1.ResourceCPU: resource.MustParse("0.1"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU: resource.MustParse("0.5"),
					},
				},
			}},
			Containers: []v1.Container{{
				Name:            kaniko.DefaultContainerName,
				Image:           "image",
				Args:            []string{"--dockerfile", "Dockerfile", "--context", "dir:///kaniko/buildcontext", "--destination", "tag", "-v", "info"},
				ImagePullPolicy: v1.PullIfNotPresent,
				Env: []v1.EnvVar{{
					Name:  "UPSTREAM_CLIENT_TYPE",
					Value: "UpstreamClient(skaffold-)",
				}, {
					Name:  "KEY",
					Value: "VALUE",
				}, {
					Name:  "HTTP_PROXY",
					Value: "http://proxy",
				}, {
					Name:  "HTTPS_PROXY",
					Value: "https://proxy",
				}, {
					Name:  "GOOGLE_APPLICATION_CREDENTIALS",
					Value: "/secret/kaniko-secret.json",
				}},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      kaniko.DefaultEmptyDirName,
						MountPath: kaniko.DefaultEmptyDirMountPath,
					},
					{
						Name:      kaniko.DefaultSecretName,
						MountPath: "/secret",
					},
					{
						Name:      "cm-volume-1",
						ReadOnly:  true,
						MountPath: "/cm-secret-mount-path",
						SubPath:   "/subpath",
					},
					{
						Name:      "secret-volume-1",
						ReadOnly:  true,
						MountPath: "/secret-secret-mount-path",
						SubPath:   "/subpath",
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: map[v1.ResourceName]resource.Quantity{
						v1.ResourceCPU: resource.MustParse("0.1"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU: resource.MustParse("0.5"),
					},
				},
			}},
			ServiceAccountName: "aVerySpecialSA",
			SecurityContext: &v1.PodSecurityContext{
				RunAsUser: &runAsUser,
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: kaniko.DefaultEmptyDirName,
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: kaniko.DefaultSecretName,
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "secret",
						},
					},
				},
				{
					Name: "cm-volume-1",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "cm-1",
							},
						},
					},
				},
				{
					Name: "secret-volume-1",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "secret-1",
						},
					},
				},
			},
			Tolerations: []v1.Toleration{
				{
					Key:               "app",
					Operator:          "Equal",
					Value:             "skaffold",
					Effect:            "NoSchedule",
					TolerationSeconds: nil,
				},
			},
			NodeSelector: map[string]string{"kubernetes.io/os": "linux", "kubernetes.io/arch": "arm64"},
		},
	}

	testutil.CheckDeepEqual(t, expectedPod.Spec.Containers[0].Env, pod.Spec.Containers[0].Env)
}

func TestResourceRequirements(t *testing.T) {
	tests := []struct {
		description string
		initial     *latest.ResourceRequirements
		expected    v1.ResourceRequirements
	}{
		{
			description: "no resource specified",
			initial:     &latest.ResourceRequirements{},
			expected:    v1.ResourceRequirements{},
		},
		{
			description: "with resource specified",
			initial: &latest.ResourceRequirements{
				Requests: &latest.ResourceRequirement{
					CPU:              "0.5",
					Memory:           "1000",
					ResourceStorage:  "1000",
					EphemeralStorage: "1000",
				},
				Limits: &latest.ResourceRequirement{
					CPU:              "1.0",
					Memory:           "2000",
					ResourceStorage:  "1000",
					EphemeralStorage: "1000",
				},
			},
			expected: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:              resource.MustParse("0.5"),
					v1.ResourceMemory:           resource.MustParse("1000"),
					v1.ResourceStorage:          resource.MustParse("1000"),
					v1.ResourceEphemeralStorage: resource.MustParse("1000"),
				},
				Limits: v1.ResourceList{
					v1.ResourceCPU:              resource.MustParse("1.0"),
					v1.ResourceMemory:           resource.MustParse("2000"),
					v1.ResourceStorage:          resource.MustParse("1000"),
					v1.ResourceEphemeralStorage: resource.MustParse("1000"),
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := resourceRequirements(test.initial)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
