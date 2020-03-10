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

package validator

import (
	"context"
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

// PodValidator implements the Validator interface for Pods
type PodValidator struct {
	k kubernetes.Interface
}

// NewPodValidator initializes a PodValidator
func NewPodValidator(k kubernetes.Interface) *PodValidator {
	return &PodValidator{k: k}
}

// Validate implements the Validate method for Validator interface
func (p *PodValidator) Validate(ctx context.Context, ns string, opts meta_v1.ListOptions) ([]Resource, error) {
	pods, err := p.k.CoreV1().Pods(ns).List(opts)
	if err != nil {
		return nil, err
	}
	rs := []Resource{}
	for _, po := range pods.Items {
		ps := p.getPodStatus(&po)
		rs = append(rs, NewResourceFromObject(&po, Status(ps.phase), ps.reason.String()))
	}
	return rs, nil
}

type podStatus struct {
	phase  string
	reason *podReason
}

type podReason struct {
	reason  string
	message string
}

func (r *podReason) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("pod unstable due to reason: %s, message: %s", r.reason, r.message)
}

func (p *PodValidator) getPodStatus(pod *v1.Pod) podStatus {
	switch pod.Status.Phase {
	case v1.PodSucceeded:
		return podStatus{phase: success}
	case v1.PodRunning:
		return podStatus{phase: running}
	default:
		return getPendingDetails(pod)
	}
}

func getPendingDetails(pod *v1.Pod) podStatus {
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

func newPendingStatus(r string, d string) podStatus {
	return podStatus{
		phase: pending,
		reason: &podReason{
			reason:  r,
			message: d,
		},
	}
}

func newUnknownStatus() podStatus {
	return podStatus{
		phase: unknown,
		reason: &podReason{
			reason:  unknown,
			message: unknown,
		},
	}
}
