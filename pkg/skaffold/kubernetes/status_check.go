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
	"fmt"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	success = "Succeeded"
	running = "Running"
	unknown = "Unknown"
	pending = "Pending"
)

// for testing
var (
	waitingContainerStatus = getWaitingContainerStatus
)

type PodStatus struct {
	phase string
	err   *PodErr
}

type PodErr struct {
	reason  string
	message string
}

func (e *PodErr) Error() string {
	return fmt.Sprintf("pod in error due to reason: %s, message: %s", e.reason, e.message)
}

func GetPodDetails(client kubernetes.Interface, ns string, podName string) PodStatus {
	pod, err := client.CoreV1().Pods(ns).Get(podName, meta_v1.GetOptions{})
	if err != nil {
		return PodStatus{err: &PodErr{message: err.Error()}}
	}
	switch pod.Status.Phase {
	case v1.PodSucceeded:
		return PodStatus{phase: success}
	case v1.PodRunning:
		return PodStatus{phase: running}
	default:
		return getPendingDetails(pod)
	}
}

func getPendingDetails(pod *v1.Pod) PodStatus {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions
	for _, c := range pod.Status.Conditions {
		switch c.Status {
		case v1.ConditionUnknown:
			return newPendingStatus(unknown, c.Message)
		default:
			// TODO(dgageot): Add EphemeralContainerStatuses
			cs := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
			reason, detail := waitingContainerStatus(cs)
			return newPendingStatus(reason, detail)
		}
	}
	return newUnknownStatus()
}

func getWaitingContainerStatus(cs []v1.ContainerStatus) (string, string) {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states
	for _, c := range cs {
		if c.State.Waiting != nil {
			return c.State.Waiting.Reason, c.State.Waiting.Message
		}
	}
	return success, success
}

func newPendingStatus(r string, d string) PodStatus {
	return PodStatus{
		phase: pending,
		err: &PodErr{
			reason:  r,
			message: d,
		},
	}
}

func newUnknownStatus() PodStatus {
	return PodStatus{
		phase: unknown,
		err: &PodErr{
			reason:  unknown,
			message: unknown,
		},
	}
}
