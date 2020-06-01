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

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestContainerManager(t *testing.T) {
	testutil.Run(t, "simulation", func(t *testutil.T) {
		startCount := 0
		terminatedCount := 0
		t.Override(&notifyDebuggingContainerStarted, func(podName string, containerName string, namespace string, artifactImage string, runtime string, workingDir string, debugPorts map[string]uint32) {
			startCount++
		})
		t.Override(&notifyDebuggingContainerTerminated, func(podName string, containerName string, namespace string, artifactImage string, runtime string, workingDir string, debugPorts map[string]uint32) {
			terminatedCount++
		})
		pod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "pod",
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
			Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{Name: "test", State: v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}}}},
		}
		m := &ContainerManager{active: make(map[string]string)}
		state := &pod.Status.ContainerStatuses[0].State

		// should never be active until running
		m.checkPod(&pod)
		t.CheckDeepEqual(0, len(m.active))
		m.checkPod(&pod)
		t.CheckDeepEqual(0, len(m.active))
		t.CheckDeepEqual(0, startCount)
		t.CheckDeepEqual(0, terminatedCount)

		// container is now running
		state.Waiting = nil
		state.Running = &v1.ContainerStateRunning{}

		m.checkPod(&pod)
		t.CheckDeepEqual(1, len(m.active))
		_, found := m.active["ns/pod/test"]
		t.CheckDeepEqual(true, found)
		t.CheckDeepEqual(1, startCount)
		t.CheckDeepEqual(0, terminatedCount)

		// container is now terminated
		state.Running = nil
		state.Terminated = &v1.ContainerStateTerminated{}

		m.checkPod(&pod)
		t.CheckDeepEqual(0, len(m.active))
		t.CheckDeepEqual(1, startCount)
		t.CheckDeepEqual(1, terminatedCount)
	})
}

func TestContainerManagerZeroValue(t *testing.T) {
	var m *ContainerManager

	// Should not raise a nil dereference
	m.Start(context.Background())
	m.Stop()
}
