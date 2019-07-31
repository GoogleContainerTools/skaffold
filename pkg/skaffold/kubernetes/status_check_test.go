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
		expected    string
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
			name:      "not-there",
			expected:  "pods \"not-there\" not found",
			shouldErr: true,
		}, {
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
							Name: "foo-containter",
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
			expected:  "container foo-containter is still waiting due to reason ErrImgPull. Detail: could not pull the container image",
		}, {
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
							Name: "foo-containter",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "UnSchedulable",
								},
							},
						},
					},
				},
			},
			shouldErr: true,
			expected:  "container foo-containter is still waiting due to reason UnSchedulable",
		}, {
			description: "pod is in Waiting condition with no reason.",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{{Status: v1.ConditionFalse}},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "foo-containter",
							State: v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}},
						},
					},
				},
			},
			expected:  "container foo-containter is still waiting due to reason",
			shouldErr: true,
		}, {
			description: "pod is in Terminated State",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{{Status: v1.ConditionTrue}},
				},
			},
		}, {
			description: "pod is in Running State",
			name:        "foo",
			pod: v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "foo",
					Namespace: "test",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{{Status: v1.ConditionTrue}},
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "foo-containter",
							State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			client := fakekubeclientset.NewSimpleClientset(&test.pod)
			fakePods := client.CoreV1().Pods("test")
			actual := GetPodDetails(fakePods, test.name)
			t.CheckError(test.shouldErr, actual)
			if actual != nil {
				t.CheckErrorContains(test.expected, actual)
			}
		})
	}

}
