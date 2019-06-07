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

package kubernetes

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"

	"k8s.io/apimachinery/pkg/watch"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func TestWaitForPodSucceeded(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "test",
		Name:      "test-pod",
	}}
	client := fakekubeclientset.NewSimpleClientset(pod)
	fakePods := client.CoreV1().Pods("test")

	// Update Pod status
	ctx := context.TODO()
	errCh := make(chan error)
	go func(ctx context.Context, pods corev1.PodInterface) {
		errCh <- WaitForPodSucceeded(ctx, fakePods, "test-pod", 10*time.Second)
	}(ctx, fakePods)

	fmt.Println("changing phase to success")
	time.Sleep(5 * time.Second)
	// pod.Status.Phase = v1.PodFailed
	pod.Status.Phase = v1.PodSucceeded
	//pod.Status.Phase = v1.PodRunning
	// fakeWatcher.Modify(pod)

	err := <-errCh
	if err != nil {
		t.Errorf("failed with %s", err)
	}
}

func TestIsPodSucceeded(t *testing.T) {

	tests := []struct {
		description string
		podName     string
		phase       v1.PodPhase
		shouldErr   bool
		expected    bool
	}{
		{
			description: "pod name doesn't match",
			podName:     "another-pod",
		}, {
			description: "pod phase is PodSucceeded",
			phase:       v1.PodSucceeded,
			expected:    true,
		}, {
			description: "pod phase is PodRunning",
			phase:       v1.PodRunning,
		}, {
			description: "pod phase is PodFailed",
			phase:       v1.PodFailed,
			shouldErr:   true,
		}, {
			description: "pod phase is PodUnknown",
			phase:       v1.PodUnknown,
		}, {
			description: "pod phase is PodPending",
			phase:       v1.PodPending,
		}, {
			description: "unknown pod phase",
			phase:       "unknownPhase",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			pod := &v1.Pod{
				Status: v1.PodStatus{
					Phase: test.phase,
				},
			}
			f := isPodSucceeded(test.podName)
			dummyEvent := &watch.Event{
				Type:   "dummyEvent",
				Object: pod,
			}
			actual, err := f(dummyEvent)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, actual, test.expected)
		})
	}
}
