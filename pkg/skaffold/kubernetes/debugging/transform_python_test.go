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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/annotations"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging/adapter"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPythonTransformer_Apply(t *testing.T) {
	tests := []struct {
		description       string
		containerSpec     v1.Container
		configuration     debug.ImageConfiguration
		overrideProtocols []string
		shouldErr         bool
		result            v1.Container
		debugConfig       annotations.ContainerDebugConfiguration
		image             string
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{},
			result:        v1.Container{},
			shouldErr:     true,
		},
		{
			description:   "basic",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"python"}},
			result: v1.Container{
				Command: []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python"},
				Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description:       "override protocol - pydevd, dap",
			containerSpec:     v1.Container{},
			configuration:     debug.ImageConfiguration{Entrypoint: []string{"python"}},
			overrideProtocols: []string{"pydevd", "dap"},
			result: v1.Container{
				Command: []string{"/dbg/python/launcher", "--mode", "pydevd", "--port", "5678", "--", "python"},
				Ports:   []v1.ContainerPort{{Name: "pydevd", ContainerPort: 5678}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"pydevd": 5678}},
			image:       "python",
		},
		{
			description:       "override protocol - dap, pydevd",
			containerSpec:     v1.Container{},
			configuration:     debug.ImageConfiguration{Entrypoint: []string{"python"}},
			overrideProtocols: []string{"dap", "pydevd"},
			result: v1.Container{
				Command: []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python"},
				Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"python"}},
			result: v1.Container{
				Command: []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python"},
				Ports:   []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}, {Name: "dap", ContainerPort: 5678}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description:   "command not entrypoint",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Arguments: []string{"python"}},
			result: v1.Container{
				Args:  []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python"},
				Ports: []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description:   "entrypoint with python env vars",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Entrypoint: []string{"foo"}, Env: map[string]string{"PYTHON_VERSION": "3"}},
			result: v1.Container{
				Command: []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "foo"},
				Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description:   "command with python env vars",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{Arguments: []string{"foo"}, Env: map[string]string{"PYTHON_VERSION": "3"}},
			result: v1.Container{
				Command: []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--"},
				Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
	}
	var identity debug.PortAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			adapter := adapter.NewAdapter(&test.containerSpec)
			config, image, err := debug.NewPythonTransformer().Apply(adapter, test.configuration, identity, test.overrideProtocols)
			adapter.Apply()

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.result, test.containerSpec)
			t.CheckDeepEqual(test.debugConfig, config)
			t.CheckDeepEqual(test.image, image)
		})
	}
}

func TestTransformManifestPython(t *testing.T) {
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
			"Pod with ptvsd",
			&v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "test",
						Command: []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
					}},
				}},
			true,
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "test",
						Command: []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
						Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
					}},
				}},
		},
		{
			"Pod with debugpy",
			&v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "test",
						Command: []string{"python", "-mdebugpy", "--listen", "5678", "foo.py"},
					}},
				}},
			true,
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "test",
						Command: []string{"python", "-mdebugpy", "--listen", "5678", "foo.py"},
						Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
					}},
				}},
		},
		{
			"Pod with Python container",
			&v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "test",
						Command: []string{"python", "foo.py"},
					}},
				}},
			true,
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:         "test",
						Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
						Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					InitContainers: []v1.Container{{
						Name:         "install-python-debug-support",
						Image:        "HELPERS/python",
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					Volumes: []v1.Volume{{
						Name:         "debugging-support-files",
						VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
					}},
				}},
		},
		{
			"Deployment with Python container",
			&appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"python", "foo.py"},
							}},
						}}}},
			true,
			&appsv1.Deployment{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-debug-support",
								Image:        "HELPERS/python",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"ReplicaSet with Python container",
			&appsv1.ReplicaSet{
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"python", "foo.py"},
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
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-debug-support",
								Image:        "HELPERS/python",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"StatefulSet with Python container",
			&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"python", "foo.py"},
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
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-debug-support",
								Image:        "HELPERS/python",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"DaemonSet with Python container",
			&appsv1.DaemonSet{
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"python", "foo.py"},
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
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-debug-support",
								Image:        "HELPERS/python",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"Job with Python container",
			&batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"python", "foo.py"},
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
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-debug-support",
								Image:        "HELPERS/python",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"ReplicationController with Python container",
			&v1.ReplicationController{
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(2),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:    "test",
								Command: []string{"python", "foo.py"},
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
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-debug-support",
								Image:        "HELPERS/python",
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							Volumes: []v1.Volume{{
								Name:         "debugging-support-files",
								VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
							}},
						}}}},
		},
		{
			"PodList with Python and non-Python container",
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
								Command: []string{"python", "foo.py"},
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
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"/dbg/python/launcher", "--mode", "debugpy", "--port", "5678", "--", "python", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-debug-support",
								Image:        "HELPERS/python",
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
