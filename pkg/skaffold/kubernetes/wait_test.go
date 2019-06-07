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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fake_testing "k8s.io/client-go/testing"
)

func TestWaitForPodComplete(t *testing.T) {
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
		errCh <- WaitForPodComplete(ctx, fakePods, "test-pod", 3*time.Second)
	}(ctx, fakePods)

	pod.Status.Phase = v1.PodFailed
	pod.Status.Phase = v1.PodSucceeded
	//pod.Status.Phase = v1.PodRunning
	fakeWatcher.Modify(pod)

	err := <-errCh
	if err != nil {
		t.Errorf("failed with %s", err)
	}

}
