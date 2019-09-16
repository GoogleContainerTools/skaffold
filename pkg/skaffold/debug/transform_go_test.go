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

package debug

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestNewDlvSpecDefaults(t *testing.T) {
	spec := newDlvSpec(20)
	expected := dlvSpec{mode: "exec", host: "localhost", port: 20, apiVersion: 2, headless: true, log: false}
	testutil.CheckDeepEqual(t, expected, spec, cmp.AllowUnexported(spec))
}

func TestExtractDlvArg(t *testing.T) {
	tests := []struct {
		in     []string
		result *dlvSpec
	}{
		{nil, nil},
		{[]string{"foo"}, nil},
		{[]string{"foo", "--foo"}, nil},
		{[]string{"dlv", "debug", "--headless"}, &dlvSpec{mode: "debug", headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "--headless", "exec"}, &dlvSpec{mode: "exec", headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "--headless", "exec", "--", "--listen=host:4345"}, &dlvSpec{mode: "exec", headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "test", "--headless", "--listen=host:4345"}, &dlvSpec{mode: "test", host: "host", port: 4345, headless: true, apiVersion: 2, log: false}},
		{[]string{"dlv", "debug", "--headless", "--api-version=1"}, &dlvSpec{mode: "debug", headless: true, apiVersion: 1, log: false}},
		{[]string{"dlv", "debug", "--listen=host:4345", "--headless", "--api-version=2", "--log"}, &dlvSpec{mode: "debug", host: "host", port: 4345, headless: true, apiVersion: 2, log: true}},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.in, " "), func(t *testing.T) {
			if test.result == nil {
				testutil.CheckDeepEqual(t, test.result, extractDlvSpec(test.in))
			} else {
				testutil.CheckDeepEqual(t, *test.result, *extractDlvSpec(test.in), cmp.AllowUnexported(dlvSpec{}))
			}
		})
	}
}

func TestDlvTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      imageConfiguration
		result      bool
	}{
		{
			description: "GOMAXPROCS",
			source:      imageConfiguration{env: map[string]string{"GOMAXPROCS": "2"}},
			result:      true,
		},
		{
			description: "GOGC",
			source:      imageConfiguration{env: map[string]string{"GOGC": "off"}},
			result:      true,
		},
		{
			description: "GODEBUG",
			source:      imageConfiguration{env: map[string]string{"GODEBUG": "efence=1"}},
			result:      true,
		},
		{
			description: "GOTRACEBACK",
			source:      imageConfiguration{env: map[string]string{"GOTRACEBACK": "off"}},
			result:      true,
		},
		{
			description: "entrypoint with dlv",
			source:      imageConfiguration{entrypoint: []string{"dlv", "exec", "--headless"}},
			result:      true,
		},
		{
			description: "entrypoint /bin/sh",
			source:      imageConfiguration{entrypoint: []string{"/bin/sh"}},
			result:      false,
		},
		{
			description: "nothing",
			source:      imageConfiguration{},
			result:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := dlvTransformer{}.IsApplicable(test.source)
			testutil.CheckDeepEqual(t, test.result, result)
		})
	}
}

func TestDlvTransformer_RuntimeSupportImage(t *testing.T) {
	testutil.CheckDeepEqual(t, "go", dlvTransformer{}.RuntimeSupportImage())
}

func TestDlvTransformerApply(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration imageConfiguration
		result        v1.Container
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{},
			result:        v1.Container{},
		},
		{
			description:   "basic",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{entrypoint: []string{"app", "arg"}},
			result: v1.Container{
				Command: []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
				Ports:   []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
			},
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: imageConfiguration{entrypoint: []string{"app", "arg"}},
			result: v1.Container{
				Command: []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
				Ports:   []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}, {Name: "dlv", ContainerPort: 56268}},
			},
		},
		{
			description:   "command not entrypoint",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{arguments: []string{"app", "arg"}},
			result: v1.Container{
				Args:  []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
				Ports: []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
			},
		},
	}
	var identity portAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			dlvTransformer{}.Apply(&test.containerSpec, test.configuration, identity)
			testutil.CheckDeepEqual(t, test.result, test.containerSpec)
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
					Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:         "test",
						Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
						Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
						Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					InitContainers: []v1.Container{{
						Name:         "install-go-support",
						Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-support",
								Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-support",
								Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-support",
								Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-support",
								Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-support",
								Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(1),
					Template: &v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-support",
								Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"dlv":56268,"runtime":"go"}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=localhost:56268", "--api-version=2", "app", "--", "arg"},
								Ports:        []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}},
								Env:          []v1.EnvVar{{Name: "GOMAXPROCS", Value: "1"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-go-support",
								Image:        "gcr.io/gcp-dev-tools/duct-tape/go",
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
		t.Run(test.description, func(t *testing.T) {
			value := test.in.DeepCopyObject()

			retriever := func(image string) (imageConfiguration, error) {
				return imageConfiguration{}, nil
			}
			result := transformManifest(value, retriever)
			testutil.CheckDeepEqual(t, test.transformed, result)
			testutil.CheckDeepEqual(t, test.out, value)
		})
	}
}
