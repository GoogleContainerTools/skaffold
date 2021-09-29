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

func TestNodeTransformer_Apply(t *testing.T) {
	// no shouldErr as Apply always succeeds
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration debug.ImageConfiguration
		result        v1.Container
		debugConfig   types.ContainerDebugConfiguration
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "NODE_OPTIONS", Value: "--inspect=0.0.0.0:9229"}, {Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "entrypoint",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "entrypoint with PATH",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"node"}, Env: map[string]string{"PATH": "/usr/bin"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin:/usr/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}, {Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description: "existing devtools port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 4444}},
			},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "command not entrypoint",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Arguments: []string{"node"}},
			result: v1.Container{
				Args:  []string{"node", "--inspect=0.0.0.0:9229"},
				Env:   []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "docker-entrypoint (#3821)",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{
				Env:        map[string]string{"NODE_VERSION": "10.12"},
				Entrypoint: []string{"docker-entrypoint.sh"},
				Arguments:  []string{"npm run script"}},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "NODE_OPTIONS", Value: "--inspect=0.0.0.0:9229"}, {Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "image environment not copied",
			containerSpec: v1.Container{Env: []v1.EnvVar{{Name: "OTHER", Value: "VALUE"}}},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"node"}, Env: map[string]string{"RANDOM": "VALUE"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "OTHER", Value: "VALUE"}, {Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: types.ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
	}
	var identity debug.PortAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			adapter := adapter.NewAdapter(&test.containerSpec)
			config, image, err := debug.NewNodeTransformer().Apply(adapter, test.configuration, identity, nil)
			adapter.Apply()

			// Apply never fails since there's always the option to set NODE_OPTIONS
			t.CheckNil(err)
			t.CheckDeepEqual(test.result, test.containerSpec)
			t.CheckDeepEqual(test.debugConfig, config)
			t.CheckDeepEqual("nodejs", image)
		})
	}
}

func TestTransformManifestNodeJS(t *testing.T) {
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
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:    "test",
					Command: []string{"echo", "Hello World"},
				}}}},
			false,
			&v1.Pod{
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:    "test",
					Command: []string{"echo", "Hello World"},
				}}}},
		},
		{
			"Pod with NodeJS container",
			&v1.Pod{
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:    "test",
					Command: []string{"node", "foo.js"},
				}}}},
			true,
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:         "test",
						Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
						Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
						Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					InitContainers: []v1.Container{{
						Name:         "install-nodejs-debug-support",
						Image:        "HELPERS/nodejs",
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					Volumes: []v1.Volume{{
						Name:         "debugging-support-files",
						VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
					}},
				}},
		},
		{
			"Deployment with NodeJS container",
			&appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{{
							Name:    "test",
							Command: []string{"node", "foo.js"},
						}}}}}},
			true,
			&appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
								Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
								Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-nodejs-debug-support",
								Image:        "HELPERS/nodejs",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"ReplicaSet with NodeJS container",
			&appsv1.ReplicaSet{
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{{
							Name:    "test",
							Command: []string{"node", "foo.js"},
						}}}}}},
			true,
			&appsv1.ReplicaSet{
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
								Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
								Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-nodejs-debug-support",
								Image:        "HELPERS/nodejs",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"StatefulSet with NodeJS container",
			&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{{
							Name:    "test",
							Command: []string{"node", "foo.js"},
						}}}}}},
			true,
			&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
								Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
								Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-nodejs-debug-support",
								Image:        "HELPERS/nodejs",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"DaemonSet with NodeJS container",
			&appsv1.DaemonSet{
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{{
							Name:    "test",
							Command: []string{"node", "foo.js"},
						}}}}}},
			true,
			&appsv1.DaemonSet{
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
								Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
								Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-nodejs-debug-support",
								Image:        "HELPERS/nodejs",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"Job with NodeJS container",
			&batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"node", "foo.js"},
							}}}}}},
			true,
			&batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
								Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
								Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-nodejs-debug-support",
								Image:        "HELPERS/nodejs",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"ReplicationController with NodeJS container",
			&v1.ReplicationController{
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(2),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"node", "foo.js"},
							},
						}}}}},
			true,
			&v1.ReplicationController{
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(1),
					Template: &v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
								Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
								Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-nodejs-debug-support",
								Image:        "HELPERS/nodejs",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"PodList with Java and non-Java container",
			&v1.PodList{
				Items: []v1.Pod{
					{
						Spec: v1.PodSpec{Containers: []v1.Container{{
							Name:    "echo",
							Command: []string{"echo", "Hello World"},
						}}},
					},
					{
						Spec: v1.PodSpec{Containers: []v1.Container{{
							Name:    "test",
							Command: []string{"node", "foo.js"},
						}}},
					},
				}},
			true,
			&v1.PodList{
				Items: []v1.Pod{
					{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "echo",
								Command: []string{"echo", "Hello World"},
							}}},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"nodejs","ports":{"devtools":9229}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"node", "--inspect=0.0.0.0:9229", "foo.js"},
								Ports:        []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
								Env:          []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-nodejs-debug-support",
								Image:        "HELPERS/nodejs",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}}},
					}}},
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
