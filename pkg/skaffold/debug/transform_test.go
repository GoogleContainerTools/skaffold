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
	"reflect"
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

func TestAllocatePort(t *testing.T) {
	// helper function to create a container
	containerWithPorts := func(ports ...int32) v1.Container {
		var created []v1.ContainerPort
		for _, port := range ports {
			created = append(created, v1.ContainerPort{ContainerPort: port})
		}
		return v1.Container{Ports: created}
	}

	tests := []struct {
		description string
		pod         v1.PodSpec
		desiredPort int32
		result      int32
	}{
		{
			description: "simple",
			pod:         v1.PodSpec{},
			desiredPort: 5005,
			result:      5005,
		},
		{
			description: "finds next available port",
			pod: v1.PodSpec{Containers: []v1.Container{
				containerWithPorts(5005, 5007),
				containerWithPorts(5008, 5006),
			}},
			desiredPort: 5005,
			result:      5009,
		},
		{
			description: "skips reserved",
			pod:         v1.PodSpec{},
			desiredPort: 1,
			result:      1024,
		},
		{
			description: "skips 0",
			pod:         v1.PodSpec{},
			desiredPort: 0,
			result:      1024,
		},
		{
			description: "skips negative",
			pod:         v1.PodSpec{},
			desiredPort: -1,
			result:      1024,
		},
		{
			description: "wraps at maxPort",
			pod:         v1.PodSpec{},
			desiredPort: 65536,
			result:      1024,
		},
		{
			description: "wraps beyond maxPort",
			pod:         v1.PodSpec{},
			desiredPort: 65537,
			result:      1024,
		},
		{
			description: "looks backwards at 65535",
			pod: v1.PodSpec{Containers: []v1.Container{
				containerWithPorts(65535),
			}},
			desiredPort: 65535,
			result:      65534,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := allocatePort(&test.pod, test.desiredPort)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		in     runtime.Object
		result string
	}{
		{&v1.Pod{TypeMeta: metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.String(), Kind: "Pod"}, ObjectMeta: metav1.ObjectMeta{Name: "name"}}, "pod/name"},
		{&v1.ReplicationController{TypeMeta: metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.String(), Kind: "ReplicationController"}, ObjectMeta: metav1.ObjectMeta{Name: "name"}}, "replicationcontroller/name"},
		{&appsv1.Deployment{TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"}, ObjectMeta: metav1.ObjectMeta{Name: "name"}}, "deployment.apps/name"},
		{&appsv1.ReplicaSet{TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "ReplicaSet"}, ObjectMeta: metav1.ObjectMeta{Name: "name"}}, "replicaset.apps/name"},
		{&appsv1.StatefulSet{TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "StatefulSet"}, ObjectMeta: metav1.ObjectMeta{Name: "name"}}, "statefulset.apps/name"},
		{&appsv1.DaemonSet{TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "DaemonSet"}, ObjectMeta: metav1.ObjectMeta{Name: "name"}}, "daemonset.apps/name"},
		{&batchv1.Job{TypeMeta: metav1.TypeMeta{APIVersion: batchv1.SchemeGroupVersion.String(), Kind: "Job"}, ObjectMeta: metav1.ObjectMeta{Name: "name"}}, "job.batch/name"},
	}
	for _, test := range tests {
		testutil.Run(t, reflect.TypeOf(test.in).Name(), func(t *testutil.T) {
			gvk := test.in.GetObjectKind().GroupVersionKind()
			group, version, kind, description := describe(test.in)

			t.CheckDeepEqual(gvk.Group, group)
			t.CheckDeepEqual(gvk.Kind, kind)
			t.CheckDeepEqual(gvk.Version, version)
			t.CheckDeepEqual(test.result, description)
		})
	}
}

func TestExposePort(t *testing.T) {
	tests := []struct {
		description string
		in          []v1.ContainerPort
		expected    []v1.ContainerPort
	}{
		{"no ports", []v1.ContainerPort{}, []v1.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"existing port", []v1.ContainerPort{{Name: "name", ContainerPort: 5555}}, []v1.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"add new port", []v1.ContainerPort{{Name: "foo", ContainerPort: 4444}}, []v1.ContainerPort{{Name: "foo", ContainerPort: 4444}, {Name: "name", ContainerPort: 5555}}},
		{"clashing port name", []v1.ContainerPort{{Name: "name", ContainerPort: 4444}}, []v1.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"clashing port value", []v1.ContainerPort{{Name: "foo", ContainerPort: 5555}}, []v1.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"clashing port name and value", []v1.ContainerPort{{ContainerPort: 5555}, {Name: "name", ContainerPort: 4444}}, []v1.ContainerPort{{Name: "name", ContainerPort: 5555}}},
		{"clashing port name and value", []v1.ContainerPort{{Name: "name", ContainerPort: 4444}, {ContainerPort: 5555}}, []v1.ContainerPort{{Name: "name", ContainerPort: 5555}}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := exposePort(test.in, "name", 5555)
			t.CheckDeepEqual(test.expected, result)
			t.CheckDeepEqual([]v1.ContainerPort{{Name: "name", ContainerPort: 5555}}, filter(result, func(p v1.ContainerPort) bool { return p.Name == "name" }))
			t.CheckDeepEqual([]v1.ContainerPort{{Name: "name", ContainerPort: 5555}}, filter(result, func(p v1.ContainerPort) bool { return p.ContainerPort == 5555 }))
		})
	}
}

func filter(ports []v1.ContainerPort, predicate func(v1.ContainerPort) bool) []v1.ContainerPort {
	var selected []v1.ContainerPort
	for _, p := range ports {
		if predicate(p) {
			selected = append(selected, p)
		}
	}
	return selected
}

func TestSetEnvVar(t *testing.T) {
	tests := []struct {
		description string
		in          []v1.EnvVar
		expected    []v1.EnvVar
	}{
		{"no entry", []v1.EnvVar{}, []v1.EnvVar{{Name: "name", Value: "new-text"}}},
		{"add new entry", []v1.EnvVar{{Name: "foo", Value: "bar"}}, []v1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "name", Value: "new-text"}}},
		{"replace existing entry", []v1.EnvVar{{Name: "name", Value: "value"}}, []v1.EnvVar{{Name: "name", Value: "new-text"}}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := setEnvVar(test.in, "name", "new-text")
			t.CheckDeepEqual(test.expected, result)
		})
	}
}

func TestShJoin(t *testing.T) {
	tests := []struct {
		in     []string
		result string
	}{
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a b"}, `"a b"`},
		{[]string{`a"b`}, `"a\"b"`},
		{[]string{`a"b`}, `"a\"b"`},
		{[]string{"a", `a"b`, "b c"}, `a "a\"b" "b c"`},
	}
	for _, test := range tests {
		testutil.Run(t, strings.Join(test.in, " "), func(t *testutil.T) {
			result := shJoin(test.in)
			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestIsEntrypointLauncher(t *testing.T) {
	tests := []struct {
		description string
		entrypoint  []string
		expected    bool
	}{
		{"nil", nil, false},
		{"expected case", []string{"launcher"}, true},
		{"launchers do not take args", []string{"launcher", "bar"}, false},
		{"non-launcher", []string{"/bin/sh"}, false},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&entrypointLaunchers, []string{"launcher"})
			result := isEntrypointLauncher(test.entrypoint)
			t.CheckDeepEqual(test.expected, result)
		})
	}
}

func TestUpdateForShDashC(t *testing.T) {
	// This test uses a transformer that reverses the entrypoint.  As a result:
	//  - any "/bin/sh -c script" style command-line should see only the script portion reversed
	//  - any non-"/bin/sh -c" command-line should have its entrypoint reversed
	tests := []struct {
		description string
		input       imageConfiguration
		unwrapped   imageConfiguration
		expected    v1.Container
	}{
		{description: "empty"},
		{
			description: "no unwrapping: entrypoint ['a', 'b']",
			input:       imageConfiguration{entrypoint: []string{"a", "b"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"a", "b"}},
			expected:    v1.Container{Command: []string{"b", "a"}},
		},
		{
			description: "no unwrapping: args ['d', 'e', 'f']",
			input:       imageConfiguration{arguments: []string{"d", "e", "f"}},
			unwrapped:   imageConfiguration{arguments: []string{"d", "e", "f"}},
		},
		{
			description: "no unwrapping: entrypoint ['a', 'b'], args [d]",
			input:       imageConfiguration{entrypoint: []string{"a", "b"}, arguments: []string{"d"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"a", "b"}, arguments: []string{"d"}},
			expected:    v1.Container{Command: []string{"b", "a"}},
		},
		{
			description: "no unwrapping: entrypoint ['/bin/sh', '-x'] (only `-c`)",
			input:       imageConfiguration{entrypoint: []string{"/bin/sh", "-x"}, arguments: []string{"d"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"/bin/sh", "-x"}, arguments: []string{"d"}},
			expected:    v1.Container{Command: []string{"-x", "/bin/sh"}},
		},
		{
			description: "no unwrapping: entrypoint ['sh', '-c', 'foo'] (not /bin/sh)",
			input:       imageConfiguration{entrypoint: []string{"sh", "-c"}, arguments: []string{"d"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"sh", "-c"}, arguments: []string{"d"}},
			expected:    v1.Container{Command: []string{"-c", "sh"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/sh', '-c', 'cmd']",
			input:       imageConfiguration{entrypoint: []string{"/bin/sh", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    v1.Container{Command: []string{"/bin/sh", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/sh', '-c'], args ['d e f']",
			input:       imageConfiguration{entrypoint: []string{"/bin/sh", "-c"}, arguments: []string{"d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    v1.Container{Args: []string{"f e d"}},
		},
		{
			description: "unwwrapped: args ['/bin/sh', '-c', 'd e f']",
			input:       imageConfiguration{arguments: []string{"/bin/sh", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    v1.Container{Args: []string{"/bin/sh", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/bash', '-c', 'd e f']",
			input:       imageConfiguration{entrypoint: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    v1.Container{Command: []string{"/bin/bash", "-c", "f e d"}},
		},
		{
			description: "entrypoint ['/bin/bash','-c'], args ['d e f']",
			input:       imageConfiguration{entrypoint: []string{"/bin/bash", "-c"}, arguments: []string{"d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    v1.Container{Args: []string{"f e d"}},
		},
		{
			description: "unwwrapped: args ['/bin/bash','-c','d e f']",
			input:       imageConfiguration{arguments: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    v1.Container{Args: []string{"/bin/bash", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint-launcher and args ['/bin/sh','-c','d e f']",
			input:       imageConfiguration{entrypoint: []string{"launcher"}, arguments: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    v1.Container{Args: []string{"/bin/bash", "-c", "f e d"}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&entrypointLaunchers, []string{"launcher"})

			container := v1.Container{}
			// The transformer reverses the unwrapped entrypoint which should be reflected into the container.Entrypoint
			updateForShDashC(&container, test.input,
				func(c *v1.Container, result imageConfiguration) (ContainerDebugConfiguration, string, error) {
					t.CheckDeepEqual(test.unwrapped, result, cmp.AllowUnexported(imageConfiguration{}))
					if len(result.entrypoint) > 0 {
						c.Command = make([]string, len(result.entrypoint))
						for i, s := range result.entrypoint {
							c.Command[len(result.entrypoint)-i-1] = s
						}
					}
					return ContainerDebugConfiguration{}, "image", nil
				})
			t.CheckDeepEqual(test.expected, container)
		})
	}
}
