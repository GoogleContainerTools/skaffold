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
	"regexp"
	"strings"
)

const (
	success = "Succeeded"
	running = "Running"
	actionableMessage = `could not determine pod status. Try kubectl describe -n %s po/%s`
	errorPrefix = `(?P<Prefix>)(?P<DaemonLog>Error response from daemon\:)(?P<Error>.*)`
	crashLoopBackOff = "CrashLoopBackOff"
	runContainerError = "RunContainerError"
	imagePullErr = "ErrImagePull"
	errImagePullBackOff = "ErrImagePullBackOff"
	containerCreating = "ContainerCreating"
)

var (
	re = regexp.MustCompile(errorPrefix)
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
		rs = append(rs, NewResourceFromObject(&po, Status(ps.phase), ps.err))
	}
	return rs, nil
}


func (p *PodValidator) getPodStatus(pod *v1.Pod) *podStatus {
	ps := newPodStatus(pod.Name, pod.Namespace, string(pod.Status.Phase))
	switch pod.Status.Phase {
	case v1.PodSucceeded:
		return ps
	default:
		return ps.withErr(getContainerStatus(pod))
	}
}

func getContainerStatus(pod *v1.Pod) error {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions
	for _, c := range pod.Status.Conditions {
		switch c.Type {
		case v1.PodScheduled:
			if c.Status == v1.ConditionFalse {
				return getTolerationsDetails(pod)
			} else if c.Status == v1.ConditionTrue {
				// TODO(dgageot): Add EphemeralContainerStatuses
				cs := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
				return getWaitingContainerStatus(cs)
			} else if c.Status == v1.ConditionUnknown {
				return fmt.Errorf(c.Message)
			}
		}
	}
	return nil
}


func getWaitingContainerStatus(cs []v1.ContainerStatus) error {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states
	for _, c := range cs {
		if c.State.Waiting != nil {
			return extractErrorMessageFromContainerStatus(c)
		}
	}
	// No waiting containers, pod should be in good health.
	return nil
}

func getTolerationsDetails(pod *v1.Pod) error {
	if len(pod.Spec.Tolerations) == 0 {
		return fmt.Errorf("%s: %s", pod.Status.Reason, pod.Status.Message)
	}
	messages := make([]string, len(pod.Spec.Tolerations))
	for i, t := range pod.Spec.Tolerations {
		if t.Effect
		switch t.Key {
		case v1.TaintNodeMemoryPressure:
			messages[i] = "1 node has Memory Pressure"
		case v1.TaintNodeDiskPressure:
			messages[i] = "1 node has Disk Pressure"
		case v1.TaintNodeNotReady:
			messages[i] = "1 node is not ready"
		case v1.TaintNodeUnreachable:
			messages[i] = "1 node is unreachable"
		case v1.TaintNodeUnschedulable:
			messages[i] = "1 node is unschedulable"
		case v1.TaintNodeNetworkUnavailable:
			messages[i] = "1 node's network not available"
		case v1.TaintNodePIDPressure:
			messages[i] = "1 node has PID pressure"
		}
	}
	return fmt.Errorf("%s: 0/%d nodes available: %s", pod.Status.Reason, len(messages), strings.Join(messages, "\n    -"))
}

type podStatus struct {
	name    string
	namespace string
	phase   string
	err     error
}

func (p *podStatus) isStable() bool {
	return p.phase == success || ( p.phase == running && p.err == nil)
}

func (p *podStatus) withErr(err error) *podStatus {
	p.err = err
	return p
}

func (p *podStatus) String() string {
	switch {
	case p.isStable():
		return ""
	default:
		if p.err != nil {
			return fmt.Sprintf("%s", p.err)
		}
	}
	return fmt.Sprintf(actionableMessage, p.namespace, p.name)
}

func extractErrorMessageFromContainerStatus(c v1.ContainerStatus) error {
	// Extract meaning full error out of container statuses.
	switch c.State.Waiting.Reason {
	case containerCreating:
		return fmt.Errorf("creating container %s", c.Name)
	case crashLoopBackOff:
		return fmt.Errorf("restarting failed container %s", c.Name)
	case imagePullErr, errImagePullBackOff:
		return fmt.Errorf("container %s is waiting to start: image %s can't be pulled", c.Name, c.Image)
	case runContainerError:
		match := re.FindStringSubmatch(c.State.Waiting.Message)
		if len(match) != 0 {
			return fmt.Errorf("container %s in error: %s", c.Name, trimSpace(match[3]))
		}
	}
	return fmt.Errorf("container %s in error: %s", c.Name, trimSpace(c.State.Waiting.Message))
}


func newPodStatus(n string, ns string, p string) *podStatus {
	return &podStatus{
		name:    n,
		namespace: ns,
		phase:   p,
	}
}


func trimSpace(msg string) string {
	return strings.Trim(msg, " ")
}