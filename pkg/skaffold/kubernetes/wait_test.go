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
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"

	"k8s.io/apimachinery/pkg/watch"

	v1 "k8s.io/api/core/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestWaitForPodSucceeded(t *testing.T) {
	tests := []struct {
		description string
		phases      []v1.PodPhase
		shouldErr   bool
	}{
		{
			description: "pod eventually succeeds",
			phases:      []v1.PodPhase{v1.PodRunning, v1.PodSucceeded},
		}, {
			description: "pod eventually fails",
			phases:      []v1.PodPhase{v1.PodRunning, v1.PodFailed},
			shouldErr:   true,
		}, {
			description: "pod times out",
			phases:      []v1.PodPhase{v1.PodRunning, v1.PodRunning, v1.PodRunning, v1.PodRunning, v1.PodRunning, v1.PodRunning},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			pod := &v1.Pod{}
			client := fakekubeclientset.NewSimpleClientset(pod)

			fakeWatcher := watch.NewRaceFreeFake()
			client.PrependWatchReactor("*", testutil.SetupFakeWatcher(fakeWatcher))
			fakePods := client.CoreV1().Pods("")

			errChan := make(chan error)
			go func() {
				errChan <- WaitForPodSucceeded(context.TODO(), fakePods, "", 5*time.Second)
			}()

			for _, phase := range test.phases {
				if fakeWatcher.IsStopped() {
					break
				}
				fakeWatcher.Modify(&v1.Pod{
					Status: v1.PodStatus{
						Phase: phase,
					},
				})
				time.Sleep(time.Second)
			}
			err := <-errChan
			testutil.CheckError(t, test.shouldErr, err)
		})
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
			dummyEvent := &watch.Event{
				Type:   "dummyEvent",
				Object: pod,
			}
			actual, err := isPodSucceeded(test.podName)(dummyEvent)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, actual, test.expected)
		})
	}
}
