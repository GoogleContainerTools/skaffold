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
	"time"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/annotations"
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
			ports := k8sPortsToContainerPorts(test.in)
			ports = exposePort(ports, "name", 5555)
			result := containerPortsToK8sPorts(ports)
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
			env := k8sEnvToContainerEnv(test.in)
			env = setEnvVar(env, "name", "new-text")
			result := containerEnvToK8sEnv(env)
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
		{[]string{"a", "b'c'd"}, `a "b'c'd"`},
		{[]string{"a", "b()"}, `a "b()"`},
		{[]string{"a", "b[]"}, `a "b[]"`},
		{[]string{"a", "b{}"}, `a "b{}"`},
		{[]string{"a", "$PORT", "${PORT}", "a ${PORT} and $PORT"}, `a $PORT "${PORT}" "a ${PORT} and $PORT"`},
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
		expected    operableContainer
	}{
		{description: "empty"},
		{
			description: "no unwrapping: entrypoint ['a', 'b']",
			input:       imageConfiguration{entrypoint: []string{"a", "b"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"a", "b"}},
			expected:    operableContainer{Command: []string{"b", "a"}},
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
			expected:    operableContainer{Command: []string{"b", "a"}},
		},
		{
			description: "no unwrapping: entrypoint ['/bin/sh', '-x'] (only `-c`)",
			input:       imageConfiguration{entrypoint: []string{"/bin/sh", "-x"}, arguments: []string{"d"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"/bin/sh", "-x"}, arguments: []string{"d"}},
			expected:    operableContainer{Command: []string{"-x", "/bin/sh"}},
		},
		{
			description: "no unwrapping: entrypoint ['sh', '-c', 'foo'] (not /bin/sh)",
			input:       imageConfiguration{entrypoint: []string{"sh", "-c"}, arguments: []string{"d"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"sh", "-c"}, arguments: []string{"d"}},
			expected:    operableContainer{Command: []string{"-c", "sh"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/sh', '-c', 'cmd']",
			input:       imageConfiguration{entrypoint: []string{"/bin/sh", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    operableContainer{Command: []string{"/bin/sh", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/sh', '-c'], args ['d e f']",
			input:       imageConfiguration{entrypoint: []string{"/bin/sh", "-c"}, arguments: []string{"d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    operableContainer{Args: []string{"f e d"}},
		},
		{
			description: "unwwrapped: args ['/bin/sh', '-c', 'd e f']",
			input:       imageConfiguration{arguments: []string{"/bin/sh", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    operableContainer{Args: []string{"/bin/sh", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint ['/bin/bash', '-c', 'd e f']",
			input:       imageConfiguration{entrypoint: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    operableContainer{Command: []string{"/bin/bash", "-c", "f e d"}},
		},
		{
			description: "entrypoint ['/bin/bash','-c'], args ['d e f']",
			input:       imageConfiguration{entrypoint: []string{"/bin/bash", "-c"}, arguments: []string{"d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    operableContainer{Args: []string{"f e d"}},
		},
		{
			description: "unwwrapped: args ['/bin/bash','-c','d e f']",
			input:       imageConfiguration{arguments: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    operableContainer{Args: []string{"/bin/bash", "-c", "f e d"}},
		},
		{
			description: "unwwrapped: entrypoint-launcher and args ['/bin/sh','-c','d e f']",
			input:       imageConfiguration{entrypoint: []string{"launcher"}, arguments: []string{"/bin/bash", "-c", "d e f"}},
			unwrapped:   imageConfiguration{entrypoint: []string{"d", "e", "f"}},
			expected:    operableContainer{Args: []string{"/bin/bash", "-c", "f e d"}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&entrypointLaunchers, []string{"launcher"})

			container := operableContainer{}
			// The transformer reverses the unwrapped entrypoint which should be reflected into the container.Entrypoint
			updateForShDashC(&container, test.input,
				func(c *operableContainer, result imageConfiguration) (annotations.ContainerDebugConfiguration, string, error) {
					t.CheckDeepEqual(test.unwrapped, result, cmp.AllowUnexported(imageConfiguration{}))
					if len(result.entrypoint) > 0 {
						c.Command = make([]string, len(result.entrypoint))
						for i, s := range result.entrypoint {
							c.Command[len(result.entrypoint)-i-1] = s
						}
					}
					return annotations.ContainerDebugConfiguration{}, "image", nil
				})
			t.CheckDeepEqual(test.expected, container)
		})
	}
}

func TestRewriteHTTPGetProbe(t *testing.T) {
	tests := []struct {
		description string
		input       v1.Probe
		minTimeout  time.Duration
		changed     bool
		expected    v1.Probe
	}{
		{
			description: "non-http probe should be skipped",
			input:       v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 10},
			minTimeout:  20 * time.Second,
			changed:     false,
		},
		{
			description: "http probe with big timeout should be skipped",
			input:       v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 100 * 60},
			minTimeout:  20 * time.Second,
			changed:     false,
		},
		{
			description: "http probe with no timeout",
			input:       v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"echo"}}}},
			minTimeout:  20 * time.Second,
			changed:     true,
			expected:    v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 20},
		},
		{
			description: "http probe with small timeout",
			input:       v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 60},
			minTimeout:  100 * time.Second,
			changed:     true,
			expected:    v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 100},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			p := test.input
			if rewriteHTTPGetProbe(&p, test.minTimeout) {
				t.CheckDeepEqual(test.expected, p)
			} else {
				t.CheckDeepEqual(test.input, p) // should not have changed
			}
		})
	}
}

// TestRewriteProbes verifies that rewriteProbes skips podspecs that have a
// `debug.cloud.google.com/config` annotation.
func TestRewriteProbes(t *testing.T) {
	tests := []struct {
		name    string
		input   v1.Pod
		changed bool
		result  v1.Pod
	}{
		{
			name: "skips pod missing debug annotation",
			input: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname"},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: false,
		},
		{
			name: "processes pod with debug annotation and uses default timeout",
			input: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/config": `{"name1":{"runtime":"test"}}`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: true,
			result: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/config": `{"name1":{"runtime":"test"}}`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 600}}}}},
		},
		{
			name: "skips pod with skip-probes annotation",
			input: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `skip`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: false,
		},
		{
			name: "processes pod with probes annotation with explicit timeout",
			input: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `1m`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: false,
			result: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `1m`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 60}}}}},
		},
		{
			name: "processes pod with probes annotation with invalid timeout",
			input: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `on`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: false,
			result: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `on`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 600}}}}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			pod := test.input
			result := rewriteProbes(&pod.ObjectMeta, &pod.Spec)
			t.CheckDeepEqual(test.changed, result)
			if test.changed {
				t.CheckDeepEqual(test.result, pod)
			} else {
				t.CheckDeepEqual(test.input, pod)
			}
		})
	}
}
