/*
Copyright 2021 The Skaffold Authors

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

package debugging

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging/adapter"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDlvTransformerApply(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration debug.ImageConfiguration
		shouldErr     bool
		result        v1.Container
		debugConfig   types.ContainerDebugConfiguration
		image         string
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{},
			shouldErr:     true,
		},
		{
			description:   "basic",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"app", "arg"}},
			result: v1.Container{
				Command: []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
				Ports:   []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "go", Ports: map[string]uint32{"dlv": 56268}},
			image:       "go",
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"app", "arg"}},
			result: v1.Container{
				Command: []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
				Ports:   []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}, {Name: "dlv", ContainerPort: 56268}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "go", Ports: map[string]uint32{"dlv": 56268}},
			image:       "go",
		},
		{
			description: "existing dlv port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "dlv", ContainerPort: 7896}},
			},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"app", "arg"}},
			result: v1.Container{
				Command: []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
				Ports:   []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "go", Ports: map[string]uint32{"dlv": 56268}},
			image:       "go",
		},
		{
			description:   "command not entrypoint",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Arguments: []string{"app", "arg"}},
			result: v1.Container{
				Args:  []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
				Ports: []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "go", Ports: map[string]uint32{"dlv": 56268}},
			image:       "go",
		},
		{
			description: "entrypoint with args in container spec",
			containerSpec: v1.Container{
				Args: []string{"arg1", "arg2"},
			},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"app"}},
			result: v1.Container{
				Command: []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--"},
				Args:    []string{"arg1", "arg2"},
				Ports:   []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "go", Ports: map[string]uint32{"dlv": 56268}},
			image:       "go",
		},
	}
	var identity debug.PortAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			adapter := adapter.NewAdapter(&test.containerSpec)
			config, image, err := debug.NewDlvTransformer().Apply(adapter, test.configuration, identity, nil)
			adapter.Apply()

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.result, test.containerSpec)
			t.CheckDeepEqual(test.debugConfig, config)
			t.CheckDeepEqual(test.image, image)
		})
	}
}

func TestTransformManifestDelve(t *testing.T) {
	int32p := func(x int32) *int32 { return &x }
	tests := []struct {
		description string
		in          runtime.Object
		transformed bool
		out         runtime.Object
	}{
		{
			"Pod with no transformable container",
			&v1.Pod{
				Spec: v1.PodSpec{Containers: []v1.Container{
					{
						Name:    "test",
						Command: []string{"echo", "Hello World"},
					},
				}}},
			false,
			&v1.Pod{
				Spec: v1.PodSpec{Containers: []v1.Container{
					{
						Name:    "test",
						Command: []string{"echo", "Hello World"},
					},
				}}},
		},
		{
			"Pod with Go container with GOMAXPROCS",
			&v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "test",
						Command: []string{"app", "arg"},
						Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
					}},
				}},
			true,
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:         "test",
						Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
						Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
						Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					InitContainers: []v1.Container{{
						Name:         "install-go-debug-support",
						Image:        "HELPERS/go",
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					Volumes: []v1.Volume{{
						Name:         "debugging-support-files",
						VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
					}},
				}},
		},
		{
			"Deployment with Go container",
			&appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"app", "arg"},
								Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
							}},
						}}}},
			true,
			&appsv1.Deployment{
				// ObjectMeta: metav1.ObjectMeta{
				//  Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-debug-support",
								Image:        "HELPERS/go",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"ReplicaSet with Go container",
			&appsv1.ReplicaSet{
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"app", "arg"},
								Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
							}},
						}}}},
			true,
			&appsv1.ReplicaSet{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-debug-support",
								Image:        "HELPERS/go",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"StatefulSet with Go container",
			&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"app", "arg"},
								Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
							}},
						}}}},
			true,
			&appsv1.StatefulSet{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-debug-support",
								Image:        "HELPERS/go",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"DaemonSet with Go container",
			&appsv1.DaemonSet{
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"app", "arg"},
								Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
							}},
						}}}},
			true,
			&appsv1.DaemonSet{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-debug-support",
								Image:        "HELPERS/go",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"Job with Go container",
			&batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"app", "arg"},
								Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
							}},
						}}}},
			true,
			&batchv1.Job{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-debug-support",
								Image:        "HELPERS/go",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"ReplicationController with Go container",
			&v1.ReplicationController{
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(2),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"app", "arg"},
								Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
							}},
						}}}},
			true,
			&v1.ReplicationController{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(1),
					Template: &v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-debug-support",
								Image:        "HELPERS/go",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"PodList with Go and non-Go container",
			&v1.PodList{
				Items: []v1.Pod{
					{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "echo",
								Command: []string{"echo", "Hello World"},
							}},
						}},
					{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"app", "arg"},
								Env:     []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
							}},
						}},
				}},
			true,
			&v1.PodList{
				Items: []v1.Pod{
					{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "echo",
								Command: []string{"echo", "Hello World"},
							}},
						}},
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"go","ports":{"dlv":56268}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-debug-support",
								Image:        "HELPERS/go",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}},
				}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			value := test.in.DeepCopyObject()

			retriever := func(image string) (debug.ImageConfiguration, error) {
				return debug.ImageConfiguration{}, nil
			}
			result := transformManifest(value, retriever, "HELPERS")

			t.CheckDeepEqual(test.transformed, result)
			t.CheckDeepEqual(test.out, value)
		})
	}
}
