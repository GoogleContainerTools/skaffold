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

func TestExtractInspectArg(t *testing.T) {
	tests := []struct {
		in     string
		result *inspectSpec
	}{
		{"", nil},
		{"foo", nil},
		{"--foo", nil},
		{"-inspect", nil},
		{"-inspect=9329", nil},
		{"--inspect", &inspectSpec{port: 9229, brk: false}},
		{"--inspect=9329", &inspectSpec{port: 9329, brk: false}},
		{"--inspect=:9329", &inspectSpec{port: 9329, brk: false}},
		{"--inspect=foo:9329", &inspectSpec{host: "foo", port: 9329, brk: false}},
		{"--inspect-brk", &inspectSpec{port: 9229, brk: true}},
		{"--inspect-brk=9329", &inspectSpec{port: 9329, brk: true}},
		{"--inspect-brk=:9329", &inspectSpec{port: 9329, brk: true}},
		{"--inspect-brk=foo:9329", &inspectSpec{host: "foo", port: 9329, brk: true}},
	}
	for _, test := range tests {
		testutil.Run(t, test.in, func(t *testutil.T) {
			if test.result == nil {
				t.CheckDeepEqual(test.result, extractInspectArg(test.in))
			} else {
				t.CheckDeepEqual(*test.result, *extractInspectArg(test.in), cmp.AllowUnexported(inspectSpec{}))
			}
		})
	}
}

func TestNodeTransformer_IsApplicable(t *testing.T) {
	tests := []struct {
		description string
		source      imageConfiguration
		launcher    string
		result      bool
	}{
		{
			description: "NODE_VERSION",
			source:      imageConfiguration{env: map[string]string{"NODE_VERSION": "10"}},
			result:      true,
		},
		{
			description: "NODEJS_VERSION",
			source:      imageConfiguration{env: map[string]string{"NODEJS_VERSION": "12"}},
			result:      true,
		},
		{
			description: "NODE_ENV",
			source:      imageConfiguration{env: map[string]string{"NODE_ENV": "production"}},
			result:      true,
		},
		{
			description: "entrypoint node",
			source:      imageConfiguration{entrypoint: []string{"node", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/node",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/node", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args node",
			source:      imageConfiguration{arguments: []string{"node", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/node",
			source:      imageConfiguration{arguments: []string{"/usr/bin/node", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint nodemon",
			source:      imageConfiguration{entrypoint: []string{"nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/nodemon",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, args nodemon",
			source:      imageConfiguration{arguments: []string{"nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/nodemon",
			source:      imageConfiguration{arguments: []string{"/usr/bin/nodemon", "init.js"}},
			result:      true,
		},
		{
			description: "entrypoint npm",
			source:      imageConfiguration{entrypoint: []string{"npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "entrypoint /usr/bin/npm",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "no entrypoint, args npm",
			source:      imageConfiguration{arguments: []string{"npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments npm",
			source:      imageConfiguration{arguments: []string{"npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "no entrypoint, arguments /usr/bin/npm",
			source:      imageConfiguration{arguments: []string{"/usr/bin/npm", "run", "dev"}},
			result:      true,
		},
		{
			description: "entrypoint /bin/sh",
			source:      imageConfiguration{entrypoint: []string{"/bin/sh"}},
			result:      false,
		},
		{
			description: "entrypoint launcher", // `node` image docker-entrypoint.sh"
			source:      imageConfiguration{entrypoint: []string{"docker-entrypoint.sh"}, arguments: []string{"npm", "run", "dev"}},
			launcher:    "docker-entrypoint.sh",
			result:      true,
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
			result := nodeTransformer{}.IsApplicable(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestRewriteNodeCommandLine(t *testing.T) {
	tests := []struct {
		in     []string
		result []string
	}{
		{[]string{"node", "index.js"}, []string{"node", "--inspect=9226", "index.js"}},
		{[]string{"node"}, []string{"node", "--inspect=9226"}},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			result := rewriteNodeCommandLine(test.in, inspectSpec{port: 9226})

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestRewriteNpmCommandLine(t *testing.T) {
	tests := []struct {
		in     []string
		result []string
	}{
		{[]string{"npm", "run", "server"}, []string{"npm", "run", "server", "--node-options=--inspect=9226"}},
		{[]string{"npm", "run", "server", "--", "option"}, []string{"npm", "run", "server", "--node-options=--inspect=9226", "--", "option"}},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			result := rewriteNpmCommandLine(test.in, inspectSpec{port: 9226})

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestNodeTransformer_Apply(t *testing.T) {
	// no shouldErr as Apply always succeeds
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration imageConfiguration
		result        v1.Container
		debugConfig   ContainerDebugConfiguration
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "NODE_OPTIONS", Value: "--inspect=0.0.0.0:9229"}, {Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "entrypoint",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "entrypoint with PATH",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{entrypoint: []string{"node"}, env: map[string]string{"PATH": "/usr/bin"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin:/usr/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: imageConfiguration{entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}, {Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description: "existing devtools port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 4444}},
			},
			configuration: imageConfiguration{entrypoint: []string{"node"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "command not entrypoint",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{arguments: []string{"node"}},
			result: v1.Container{
				Args:  []string{"node", "--inspect=0.0.0.0:9229"},
				Env:   []v1.EnvVar{{Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "docker-entrypoint (#3821)",
			containerSpec: v1.Container{},
			configuration: imageConfiguration{
				env:        map[string]string{"NODE_VERSION": "10.12"},
				entrypoint: []string{"docker-entrypoint.sh"},
				arguments:  []string{"npm run script"}},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "NODE_OPTIONS", Value: "--inspect=0.0.0.0:9229"}, {Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports: []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
		{
			description:   "image environment not copied",
			containerSpec: v1.Container{Env: []v1.EnvVar{{Name: "OTHER", Value: "VALUE"}}},
			configuration: imageConfiguration{entrypoint: []string{"node"}, env: map[string]string{"RANDOM": "VALUE"}},
			result: v1.Container{
				Command: []string{"node", "--inspect=0.0.0.0:9229"},
				Env:     []v1.EnvVar{{Name: "OTHER", Value: "VALUE"}, {Name: "PATH", Value: "/dbg/nodejs/bin"}},
				Ports:   []v1.ContainerPort{{Name: "devtools", ContainerPort: 9229}},
			},
			debugConfig: ContainerDebugConfiguration{Runtime: "nodejs", Ports: map[string]uint32{"devtools": 9229}},
		},
	}
	var identity portAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			config, image, err := nodeTransformer{}.Apply(&test.containerSpec, test.configuration, identity)

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
						Name:         "install-nodejs-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
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
								Name:         "install-nodejs-support",
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
								Name:         "install-nodejs-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
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
								Name:         "install-nodejs-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
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
								Name:         "install-nodejs-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
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
								Name:         "install-nodejs-support",
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
				//ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				//},
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
								Name:         "install-nodejs-support",
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
								Name:         "install-nodejs-support",
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

			retriever := func(image string) (imageConfiguration, error) {
				return imageConfiguration{}, nil
			}
			result := transformManifest(value, retriever, "HELPERS")

			t.CheckDeepEqual(test.transformed, result)
			t.CheckDeepEqual(test.out, value)
		})
	}
}
