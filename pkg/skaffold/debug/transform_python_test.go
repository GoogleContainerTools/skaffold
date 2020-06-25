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

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestExtractPtvsdArg(t *testing.T) {
	tests := []struct {
		in     []string
		result *ptvsdSpec
	}{
		{nil, nil},
		{[]string{"foo"}, nil},
		{[]string{"--foo"}, nil},
		{[]string{"-mfoo"}, nil},
		{[]string{"-m", "foo"}, nil},
		{[]string{"-mptvsd"}, &ptvsdSpec{port: 5678, wait: false}},
		{[]string{"-m", "ptvsd", "--port", "9329"}, &ptvsdSpec{port: 9329, wait: false}},
		{[]string{"-mptvsd", "--port", "9329", "--host", "foo"}, &ptvsdSpec{host: "foo", port: 9329, wait: false}},
		{[]string{"-mptvsd", "--wait"}, &ptvsdSpec{port: 5678, wait: true}},
		{[]string{"-m", "ptvsd", "--wait", "--port", "9329", "--host", "foo"}, &ptvsdSpec{host: "foo", port: 9329, wait: true}},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			if test.result == nil {
				t.CheckDeepEqual(test.result, extractPtvsdArg(test.in))
			} else {
				t.CheckDeepEqual(*test.result, *extractPtvsdArg(test.in), cmp.AllowUnexported(ptvsdSpec{}))
			}
		})
	}
}

func TestPythonTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      imageConfiguration
		launcher    string
		result      bool
	}{
		{
			description: "PYTHON_VERSION",
			source:      imageConfiguration{env: map[string]string{"PYTHON_VERSION": "2.7"}},
			result:      false,
		},
		{
			description: "entrypoint python",
			source:      imageConfiguration{entrypoint: []string{"python", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/python",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/python", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args python",
			source:      imageConfiguration{arguments: []string{"python", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/python",
			source:      imageConfiguration{arguments: []string{"/usr/bin/python", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint python2",
			source:      imageConfiguration{entrypoint: []string{"python2", "init.py"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/python2",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/python2", "init.py"}},
			result:      true,
		},
		{
			description: "no entrypoint, args python2",
			source:      imageConfiguration{arguments: []string{"python2", "init.py"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/python2",
			source:      imageConfiguration{arguments: []string{"/usr/bin/python2", "init.py"}},
			result:      true,
		},
		{
			description: "entrypoint launcher",
			source:      imageConfiguration{entrypoint: []string{"launcher"}, arguments: []string{"python3", "app.py"}},
			launcher:    "launcher",
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
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&entrypointLaunchers, []string{test.launcher})
			result := pythonTransformer{}.IsApplicable(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestPythonTransformer_Apply(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration imageConfiguration
		shouldErr     bool
		result        v1.Container
		debugConfig   ContainerDebugConfiguration
		image         string
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{},
			result:        v1.Container{},
			shouldErr:     true,
		},
		{
			description:   "basic",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{entrypoint: []string{"python"}},
			result: v1.Container{
				Command: []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678"},
				Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
				Env:     []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: imageConfiguration{entrypoint: []string{"python"}},
			result: v1.Container{
				Command: []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678"},
				Ports:   []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}, {Name: "dap", ContainerPort: 5678}},
				Env:     []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description: "existing port and env",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "dap", ContainerPort: 8080}},
			},
			configuration: imageConfiguration{entrypoint: []string{"python"}, env: map[string]string{"PYTHONUSERBASE": "/foo"}},
			result: v1.Container{
				Command: []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678"},
				Ports:   []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
				Env:     []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python:/foo"}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
		{
			description:   "command not entrypoint",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{arguments: []string{"python"}},
			result: v1.Container{
				Args:  []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678"},
				Ports: []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
				Env:   []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "python", Ports: map[string]uint32{"dap": 5678}},
			image:       "python",
		},
	}
	var identity portAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			config, image, err := pythonTransformer{}.Apply(&test.containerSpec, test.configuration, identity)

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
						Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
						Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
						Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
						VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
					}},
					InitContainers: []v1.Container{{
						Name:         "install-python-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(1),
					Template: &v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"python","ports":{"dap":5678}}}`},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:         "test",
								Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-support",
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
								Command:      []string{"python", "-mptvsd", "--host", "0.0.0.0", "--port", "5678", "foo.py"},
								Ports:        []v1.ContainerPort{{Name: "dap", ContainerPort: 5678}},
								Env:          []v1.EnvVar{{Name: "PYTHONUSERBASE", Value: "/dbg/python"}},
								VolumeMounts: []v1.VolumeMount{{Name: "debugging-support-files", MountPath: "/dbg"}},
							}},
							InitContainers: []v1.Container{{
								Name:         "install-python-support",
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

			retriever := func(image string) (imageConfiguration, error) {
				return imageConfiguration{}, nil
			}
			result := transformManifest(value, retriever, "HELPERS")

			t.CheckDeepEqual(test.transformed, result)
			t.CheckDeepEqual(test.out, value)
		})
	}
}
