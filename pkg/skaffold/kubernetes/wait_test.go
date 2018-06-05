/*
Copyright 2018 The Skaffold Authors

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
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var podReadyState = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: "podname",
	},
	Status: v1.PodStatus{
		Phase: v1.PodRunning,
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "container_name",
				Image: "image_name",
			},
		},
	},
}

var podUnitialized = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: "podname",
	},
	Status: v1.PodStatus{
		Conditions: []v1.PodCondition{
			{
				Type: v1.PodScheduled,
			},
		},
		Phase: v1.PodPending,
	},
}

var podBadPhase = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: "podname",
	},
	Status: v1.PodStatus{
		Conditions: []v1.PodCondition{
			{
				Type: v1.PodScheduled,
			},
		},
		Phase: "not a real phase",
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "container_name",
				Image: "image_name",
			},
		},
	},
}

func TestWaitForPodReady(t *testing.T) {
	var tests = []struct {
		description string
		initialObj  *v1.Pod
		phases      []v1.PodPhase
		timeout     time.Duration

		shouldErr bool
	}{
		{
			description: "pod already ready",
			initialObj:  podReadyState,
		},
		{
			description: "pod uninitialized to succeed without running",
			initialObj:  podUnitialized,
			phases:      []v1.PodPhase{v1.PodUnknown, v1.PodSucceeded},
			shouldErr:   true,
		},
		{
			description: "pod bad phase",
			initialObj:  podBadPhase,
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			client := fake.NewSimpleClientset(test.initialObj)
			pods := client.CoreV1().Pods("")
			errCh := make(chan error, 1)
			done := make(chan struct{}, 1)
			go func() {
				errCh <- WaitForPodReady(pods, "podname")
				done <- struct{}{}
			}()
			for _, p := range test.phases {
				time.Sleep(501 * time.Millisecond)
				test.initialObj.Status.Phase = p
				pods.UpdateStatus(test.initialObj)
			}
			<-done
			var err error
			select {
			case waitErr := <-errCh:
				err = waitErr
			default:
			}
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
