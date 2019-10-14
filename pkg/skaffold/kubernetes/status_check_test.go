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
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetPodDetails(t *testing.T) {
	tests := []struct {
		description string
		pod         v1.Pod
		name        string
		expected    PodStatus
		shouldErr   bool
	}{
		{
			description: "pod does not exist",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
			},
			name: "not-there",
			expected: PodStatus{
				err: &PodErr{
					message: "pods \"not-there\" not found",
				},
			},
			shouldErr: true,
		},
		{
			description: "pod is Waiting conditions with reason and message",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Phase:      v1.PodPending,
					Conditions: []v1.PodCondition{{Status: v1.ConditionFalse}},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "foo-container",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason:  "ErrImgPull",
									Message: "could not pull the container image",
								},
							},
						},
					},
				},
			},
			shouldErr: true,
			expected: PodStatus{
				phase: pending,
				err: &PodErr{
					reason:  "ErrImgPull",
					message: "could not pull the container image",
				},
			},
		},
		{
			description: "pod is Waiting conditions with reason but no message",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Phase:      v1.PodReasonUnschedulable,
					Conditions: []v1.PodCondition{{Status: v1.ConditionFalse}},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "foo-container",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "Unschedulable",
								},
							},
						},
					},
				},
			},
			shouldErr: true,
			expected: PodStatus{
				phase: pending,
				err: &PodErr{
					reason: "Unschedulable",
				},
			},
		},
		{
			description: "pod is in Terminated State",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Phase:      v1.PodSucceeded,
					Conditions: []v1.PodCondition{{Status: v1.ConditionTrue}},
				},
			},
			expected: PodStatus{
				phase: "Succeeded",
			},
		},
		{
			description: "pod is in Stable State",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Phase:      v1.PodRunning,
					Conditions: []v1.PodCondition{{Status: v1.ConditionTrue}},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "foo-container",
							State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
						},
					},
				},
			},
			expected: PodStatus{
				phase: running,
			},
		},
		{
			description: "pod condition unknown",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					Conditions: []v1.PodCondition{{
						Status:  v1.ConditionUnknown,
						Message: "could not determine",
					}},
				},
			},
			expected: PodStatus{
				phase: pending,
				err: &PodErr{
					reason:  "Unknown",
					message: "could not determine",
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			client := fakekubeclientset.NewSimpleClientset(&test.pod)
			actual := GetPodDetails(client, "test", test.name)
			t.CheckDeepEqual(test.expected, actual, cmp.AllowUnexported(PodStatus{}, PodErr{}))
		})
	}
}

func TestGetWaitingContainerStatus(t *testing.T) {
	tests := []struct {
		description    string
		status         []v1.ContainerStatus
		expectedReason string
		expectedDetail string
	}{
		{
			description:    "no containers at all ",
			status:         []v1.ContainerStatus{},
			expectedReason: "Succeeded",
			expectedDetail: "Succeeded",
		},
		{
			description: "none of the container status is waiting",
			status: []v1.ContainerStatus{
				{
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
				{
					State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{}},
				},
			},
			expectedReason: "Succeeded",
			expectedDetail: "Succeeded",
		},
		{
			description: "one container state waiting",
			status: []v1.ContainerStatus{
				{
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
				{
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason:  "ErrImagePull",
							Message: "Cannot pull image gcr.io/test",
						},
					},
				},
			},
			expectedReason: "ErrImagePull",
			expectedDetail: "Cannot pull image gcr.io/test",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			reason, detail := getWaitingContainerStatus(test.status)
			t.CheckDeepEqual(test.expectedReason, reason)
			t.CheckDeepEqual(test.expectedDetail, detail)
		})
	}
}

type mockCS struct {
	len int
}

func (m *mockCS) mockWaitingContainerStatus(cs []v1.ContainerStatus) (string, string) {
	m.len = len(cs)
	return "", ""
}

func TestGetPendingDetails(t *testing.T) {
	tests := []struct {
		description string
		init        []v1.ContainerStatus
		cs          []v1.ContainerStatus
		expected    int
	}{
		{
			description: "pod with init containers containers",
			init: []v1.ContainerStatus{
				{
					Name:  "foo-init",
					State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}},
				},
			},
			cs: []v1.ContainerStatus{
				{
					Name:  "foo-container",
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
			},
			expected: 2,
		},
		{
			description: "pod with only init containers",
			init: []v1.ContainerStatus{
				{
					Name:  "foo-init",
					State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}},
				},
			},
			cs: []v1.ContainerStatus{
				{
					Name:  "foo-container",
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
			},
			expected: 2,
		},
		{
			description: "pod with only containers",
			cs: []v1.ContainerStatus{
				{
					Name:  "foo-container",
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
			},
			expected: 1,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			m := mockCS{}
			t.Override(&waitingContainerStatus, m.mockWaitingContainerStatus)
			pod := &v1.Pod{
				Status: v1.PodStatus{
					Conditions:            []v1.PodCondition{{Status: v1.ConditionFalse}},
					InitContainerStatuses: test.init,
					ContainerStatuses:     test.cs,
				},
			}
			getPendingDetails(pod)
			t.CheckDeepEqual(test.expected, m.len)
		})
	}
}
