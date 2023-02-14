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
	"reflect"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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
			portAvailable := func(port int32) bool {
				return isPortAvailable(&test.pod, port)
			}
			result := util.AllocatePort(portAvailable, test.desiredPort)

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
			group, version, kind, description := Describe(test.in)

			t.CheckDeepEqual(gvk.Group, group)
			t.CheckDeepEqual(gvk.Kind, kind)
			t.CheckDeepEqual(gvk.Version, version)
			t.CheckDeepEqual(test.result, description)
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
			input:       v1.Probe{ProbeHandler: v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 10},
			minTimeout:  20 * time.Second,
			changed:     false,
		},
		{
			description: "http probe with big timeout should be skipped",
			input:       v1.Probe{ProbeHandler: v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 100 * 60},
			minTimeout:  20 * time.Second,
			changed:     false,
		},
		{
			description: "http probe with no timeout",
			input:       v1.Probe{ProbeHandler: v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"echo"}}}},
			minTimeout:  20 * time.Second,
			changed:     true,
			expected:    v1.Probe{ProbeHandler: v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 20},
		},
		{
			description: "http probe with small timeout",
			input:       v1.Probe{ProbeHandler: v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 60},
			minTimeout:  100 * time.Second,
			changed:     true,
			expected:    v1.Probe{ProbeHandler: v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"echo"}}}, TimeoutSeconds: 100},
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
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
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
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: true,
			result: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/config": `{"name1":{"runtime":"test"}}`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 600}}}}},
		},
		{
			name: "skips pod with skip-probes annotation",
			input: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `skip`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
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
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: false,
			result: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `1m`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 60}}}}},
		},
		{
			name: "processes pod with probes annotation with invalid timeout",
			input: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `on`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 1}}}}},
			changed: false,
			result: v1.Pod{
				TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/probe/timeouts": `on`}},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:          "name1",
					Image:         "image1",
					LivenessProbe: &v1.Probe{ProbeHandler: v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/", Port: intstr.FromInt(8080)}}, TimeoutSeconds: 600}}}}},
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
