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

package debugging

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestContainerManager(t *testing.T) {
	makePod := func(name string, state v1.ContainerState) v1.Pod {
		return v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   "ns",
				Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"jvm","debugPorts":{"jdwp":5005}}}`},
			},
			Spec: v1.PodSpec{Containers: []v1.Container{
				{
					Name:    "test",
					Command: []string{"java", "-jar", "foo.jar"},
					Env:     []v1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"}},
					Ports:   []v1.ContainerPort{{Name: "jdwp", ContainerPort: 5005}},
				}}},
			Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{Name: "test", State: state}}},
		}
	}

	type podEvent struct {
		eventType watch.EventType
		pod       v1.Pod

		wantActiveKeys      []string
		wantStartCount      int
		wantTerminatedCount int
	}
	tests := []struct {
		description string
		events      []podEvent
	}{
		{
			description: "pod added, container started, and then terminates",
			events: []podEvent{
				{eventType: watch.Added, pod: makePod("pod", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Modified, pod: makePod("pod", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Modified, pod: makePod("pod", v1.ContainerState{Running: &v1.ContainerStateRunning{}}), wantStartCount: 1, wantActiveKeys: []string{"ns/pod/test"}},
				{eventType: watch.Modified, pod: makePod("pod", v1.ContainerState{Running: &v1.ContainerStateRunning{}}), wantStartCount: 1, wantActiveKeys: []string{"ns/pod/test"}},
				{eventType: watch.Modified, pod: makePod("pod", v1.ContainerState{Terminated: &v1.ContainerStateTerminated{}}), wantStartCount: 1, wantTerminatedCount: 1},
				{eventType: watch.Deleted, pod: makePod("pod", v1.ContainerState{Terminated: &v1.ContainerStateTerminated{}}), wantStartCount: 1, wantTerminatedCount: 1},
			},
		},
		{
			description: "pod added, container started, and then deleted before termination",
			events: []podEvent{
				{eventType: watch.Added, pod: makePod("pod", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Modified, pod: makePod("pod", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Modified, pod: makePod("pod", v1.ContainerState{Running: &v1.ContainerStateRunning{}}), wantStartCount: 1, wantActiveKeys: []string{"ns/pod/test"}},
				{eventType: watch.Deleted, pod: makePod("pod", v1.ContainerState{Running: &v1.ContainerStateRunning{}}), wantStartCount: 1, wantTerminatedCount: 1},
			},
		},
		{
			description: "pods added, container started, and terminated several times (like iterative debugging)",
			events: []podEvent{
				{eventType: watch.Added, pod: makePod("pod1", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Modified, pod: makePod("pod1", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Modified, pod: makePod("pod1", v1.ContainerState{Running: &v1.ContainerStateRunning{}}), wantStartCount: 1, wantActiveKeys: []string{"ns/pod1/test"}},
				{eventType: watch.Added, pod: makePod("pod2", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 1, wantActiveKeys: []string{"ns/pod1/test"}},
				{eventType: watch.Modified, pod: makePod("pod2", v1.ContainerState{Running: &v1.ContainerStateRunning{}}), wantStartCount: 2, wantActiveKeys: []string{"ns/pod1/test", "ns/pod2/test"}},
				{eventType: watch.Modified, pod: makePod("pod1", v1.ContainerState{Terminated: &v1.ContainerStateTerminated{}}), wantStartCount: 2, wantTerminatedCount: 1, wantActiveKeys: []string{"ns/pod2/test"}},
				{eventType: watch.Deleted, pod: makePod("pod1", v1.ContainerState{Terminated: &v1.ContainerStateTerminated{}}), wantStartCount: 2, wantTerminatedCount: 1, wantActiveKeys: []string{"ns/pod2/test"}},
				{eventType: watch.Deleted, pod: makePod("pod2", v1.ContainerState{Terminated: &v1.ContainerStateTerminated{}}), wantStartCount: 2, wantTerminatedCount: 2},
			},
		},
		{
			description: "pod added, container never started, and then deleted",
			events: []podEvent{
				{eventType: watch.Added, pod: makePod("pod", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Modified, pod: makePod("pod", v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}), wantStartCount: 0},
				{eventType: watch.Deleted, pod: makePod("pod", v1.ContainerState{Terminated: &v1.ContainerStateTerminated{}}), wantStartCount: 0, wantTerminatedCount: 0},
			},
		},
	}

	for _, tc := range tests {
		testutil.Run(t, tc.description, func(t *testutil.T) {
			ev1StartCount, ev1TerminatedCount := 0, 0
			ev2StartCount, ev2TerminatedCount := 0, 0
			// Override event v1 funcs to do nothing to avoid additional overhead
			t.Override(&notifyDebuggingContainerStarted, func(podName string, containerName string, namespace string, artifactImage string, runtime string, workingDir string, debugPorts map[string]uint32) {
				ev1StartCount++
			})
			t.Override(&notifyDebuggingContainerTerminated, func(podName string, containerName string, namespace string, artifactImage string, runtime string, workingDir string, debugPorts map[string]uint32) {
				ev1TerminatedCount++
			})
			t.Override(&debuggingContainerStartedV2, func(podName string, containerName string, namespace string, artifactImage string, runtime string, workingDir string, debugPorts map[string]uint32) {
				ev2StartCount++
			})
			t.Override(&debuggingContainerTerminatedV2, func(podName string, containerName string, namespace string, artifactImage string, runtime string, workingDir string, debugPorts map[string]uint32) {
				ev2TerminatedCount++
			})

			m := &ContainerManager{active: make(map[string]string)}

			for i, event := range tc.events {
				m.checkPod(event.eventType, &event.pod)
				if len(event.wantActiveKeys) != len(m.active) {
					t.Fatalf("step %d: active pod count: got=%d want=%d: active=%v", i, len(m.active), len(event.wantActiveKeys), m.active)
				}
				if event.wantStartCount != ev1StartCount {
					t.Fatalf("step %d: v1 start count: got=%d want=%d", i, ev1StartCount, event.wantStartCount)
				}
				if event.wantTerminatedCount != ev1TerminatedCount {
					t.Fatalf("step %d: v1 terminated count: got=%d want=%d", i, ev1TerminatedCount, event.wantTerminatedCount)
				}
				if event.wantStartCount != ev2StartCount {
					t.Fatalf("step %d: v2 start count: got=%d want=%d", i, ev2StartCount, event.wantStartCount)
				}
				if event.wantTerminatedCount != ev2TerminatedCount {
					t.Fatalf("step %d: v2 terminated count: got=%d want=%d", i, ev2TerminatedCount, event.wantTerminatedCount)
				}
				for _, key := range event.wantActiveKeys {
					if _, found := m.active[key]; !found {
						t.Fatalf("step %d: expected to find pod %q in active list: got=%v want=%v", i, key, m.active, event.wantActiveKeys)
					}
				}
			}
		})
	}
}

func TestContainerManagerZeroValue(t *testing.T) {
	var m *ContainerManager

	// Should not raise a nil dereference
	m.Start(context.Background())
	m.Stop()
}
