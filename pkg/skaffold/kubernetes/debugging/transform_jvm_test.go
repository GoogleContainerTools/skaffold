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

func TestJdwpTransformerApply(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec v1.Container
		configuration debug.ImageConfiguration
		result        v1.Container
		debugConfig   annotations.ContainerDebugConfiguration
		image         string
	}{
		{
			description:   "empty",
			containerSpec: v1.Container{},
			configuration: debug.ImageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "jvm", Ports: map[string]uint32{"jdwp": 5005}},
		},
		{
			description: "existing port",
			containerSpec: v1.Container{
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}},
			},
			configuration: debug.ImageConfiguration{},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{{Name: "http-server", ContainerPort: 8080}, {Name: "jdwp", ContainerPort: 5005}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "jvm", Ports: map[string]uint32{"jdwp": 5005}},
		},
		{
			description: "existing jdwp spec",
			containerSpec: v1.Container{
				Env:   []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=8000,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{{ContainerPort: 5005}},
			},
			configuration: debug.ImageConfiguration{Env: map[string]string{"JAVA_TOOL_OPTIONS": "-agentlib:jdwp=transport=dt_socket,server=y,address=8000,suspend=n,quiet=y"}},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=8000,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{{ContainerPort: 5005}, {Name: "jdwp", ContainerPort: 8000}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "jvm", Ports: map[string]uint32{"jdwp": 8000}},
		},
		{
			description: "existing jdwp port and JAVA_TOOL_OPTIONS",
			containerSpec: v1.Container{
				Env:   []v1.EnvVar{{Name: "FOO", Value: "BAR"}},
				Ports: []v1.ContainerPort{{Name: "jdwp", ContainerPort: 8000}},
			},
			configuration: debug.ImageConfiguration{Env: map[string]string{"JAVA_TOOL_OPTIONS": "-Xms1g"}},
			result: v1.Container{
				Env:   []v1.EnvVar{{Name: "FOO", Value: "BAR"}, {Name: "JAVA_TOOL_OPTIONS", Value: "-Xms1g -agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
				Ports: []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
			},
			debugConfig: annotations.ContainerDebugConfiguration{Runtime: "jvm", Ports: map[string]uint32{"jdwp": 5005}},
		},
	}
	var identity debug.PortAllocator = func(port int32) int32 {
		return port
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			adapter := adapter.NewAdapter(&test.containerSpec)
			config, image, err := debug.NewJDWPTransformer().Apply(adapter, test.configuration, identity, nil)
			adapter.Apply()

			// Apply never fails since there's always the option to set JAVA_TOOL_OPTIONS
			t.CheckNil(err)
			t.CheckDeepEqual(test.result, test.containerSpec)
			t.CheckDeepEqual(test.debugConfig, config)
			t.CheckDeepEqual(test.image, image)
		})
	}
}

func TestTransformManifestJVM(t *testing.T) {
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
			"Pod with Java container",
			&v1.Pod{
				Spec: v1.PodSpec{Containers: []v1.Container{
					{
						Name:    "test",
						Command: []string{"java", "-jar", "foo.jar"},
					},
				}}},
			true,
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
				},
				Spec: v1.PodSpec{Containers: []v1.Container{
					{
						Name:    "test",
						Command: []string{"java", "-jar", "foo.jar"},
						Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
						Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
					},
				}}},
		},
		{
			"Deployment with Java container",
			&appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
							},
						}}}}},
			true,
			&appsv1.Deployment{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.DeploymentSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
						},
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
								Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
								Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
							},
						}}}}},
		},
		{
			"ReplicaSet with Java container",
			&appsv1.ReplicaSet{
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
							},
						}}}}},
			true,
			&appsv1.ReplicaSet{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.ReplicaSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
						},
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
								Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
								Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
							},
						}}}}},
		},
		{
			"StatefulSet with Java container",
			&appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
							},
						}}}}},
			true,
			&appsv1.StatefulSet{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32p(1),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
						},
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
								Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
								Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
							},
						}}}}},
		},
		{
			"DaemonSet with Java container",
			&appsv1.DaemonSet{
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
							},
						}}}}},
			true,
			&appsv1.DaemonSet{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
						},
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
								Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
								Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
							},
						}}}}},
		},
		{
			"Job with Java container",
			&batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
							},
						}}}}},
			true,
			&batchv1.Job{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
						},
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
								Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
								Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
							},
						}}}}},
		},
		{
			"ReplicationController with Java container",
			&v1.ReplicationController{
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(2),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
							},
						}}}}},
			true,
			&v1.ReplicationController{
				// ObjectMeta: metav1.ObjectMeta{
				//	Labels: map[string]string{"debug.cloud.google.com/enabled": `yes`},
				// },
				Spec: v1.ReplicationControllerSpec{
					Replicas: int32p(1),
					Template: &v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
						},
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
								Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
								Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
							},
						}}}}},
		},
		{
			"PodList with Java and non-Java container",
			&v1.PodList{
				Items: []v1.Pod{
					{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "echo",
								Command: []string{"echo", "Hello World"},
							},
						}}},
					{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
							},
						}}},
				}},
			true,
			&v1.PodList{
				Items: []v1.Pod{
					{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "echo",
								Command: []string{"echo", "Hello World"},
							},
						}}},
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","ports":{"jdwp":5005}}}`},
						},
						Spec: v1.PodSpec{Containers: []v1.Container{
							{
								Name:    "test",
								Command: []string{"java", "-jar", "foo.jar"},
								Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
								Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
							},
						}}},
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
