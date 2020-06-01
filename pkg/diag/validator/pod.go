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
	"regexp"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/proto"
)

const (
	success             = "Succeeded"
	running             = "Running"
	actionableMessage   = `could not determine pod status. Try kubectl describe -n %s po/%s`
	errorPrefix         = `(?P<Prefix>)(?P<DaemonLog>Error response from daemon\:)(?P<Error>.*)`
	taintsExp           = `\{(?P<taint>.*?):.*?}`
	crashLoopBackOff    = "CrashLoopBackOff"
	runContainerError   = "RunContainerError"
	imagePullErr        = "ErrImagePull"
	imagePullBackOff    = "ImagePullBackOff"
	errImagePullBackOff = "ErrImagePullBackOff"
	containerCreating   = "ContainerCreating"
	podKind             = "pod"
)

var (
	runContainerRe = regexp.MustCompile(errorPrefix)
	taintsRe       = regexp.MustCompile(taintsExp)
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
func (p *PodValidator) Validate(ctx context.Context, ns string, opts metav1.ListOptions) ([]Resource, error) {
	pods, err := p.k.CoreV1().Pods(ns).List(opts)
	if err != nil {
		return nil, err
	}

	var rs []Resource
	for _, po := range pods.Items {
		ps := p.getPodStatus(&po)
		// The GVK group is not populated for List Objects. Hence set `kind` to `pod`
		// See https://github.com/kubernetes-sigs/controller-runtime/pull/389
		if po.Kind == "" {
			po.Kind = podKind
		}
		rs = append(rs, NewResourceFromObject(&po, Status(ps.phase), ps.err, ps.statusCode))
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

func getContainerStatus(pod *v1.Pod) (proto.StatusCode, error) {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodScheduled {
			switch c.Status {
			case v1.ConditionFalse:
				return getUntoleratedTaints(c.Reason, c.Message)
			case v1.ConditionTrue:
				// TODO(dgageot): Add EphemeralContainerStatuses
				cs := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
				// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states
				return getWaitingContainerStatus(cs)
			case v1.ConditionUnknown:
				return proto.StatusCode_STATUSCHECK_UNKNOWN, fmt.Errorf(c.Message)
			}
		}
	}
	return proto.StatusCode_STATUSCHECK_SUCCESS, nil
}

func getWaitingContainerStatus(cs []v1.ContainerStatus) (proto.StatusCode, error) {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states
	for _, c := range cs {
		switch {
		case c.State.Waiting != nil:
			return extractErrorMessageFromWaitingContainerStatus(c)
		case c.State.Terminated != nil:
			// TODO Add pod logs
			return proto.StatusCode_STATUSCHECK_CONTAINER_TERMINATED, fmt.Errorf("container %s terminated with exit code %d", c.Name, c.State.Terminated.ExitCode)
		}
	}
	// No waiting or terminated containers, pod should be in good health.
	return proto.StatusCode_STATUSCHECK_SUCCESS, nil
}

func getUntoleratedTaints(reason string, message string) (proto.StatusCode, error) {
	matches := taintsRe.FindAllStringSubmatch(message, -1)
	errCode := proto.StatusCode_STATUSCHECK_UNKNOWN_UNSCHEDULABLE
	if len(matches) == 0 {
		return errCode, fmt.Errorf("%s: %s", reason, message)
	}
	messages := make([]string, len(matches))
	// TODO: Add actionable item to fix these errors.
	for i, m := range matches {
		if len(m) < 2 {
			continue
		}
		t := m[1]
		switch t {
		case v1.TaintNodeMemoryPressure:
			messages[i] = "1 node has memory pressure"
			errCode = proto.StatusCode_STATUSCHECK_NODE_MEMORY_PRESSURE
		case v1.TaintNodeDiskPressure:
			messages[i] = "1 node has disk pressure"
			errCode = proto.StatusCode_STATUSCHECK_NODE_DISK_PRESSURE
		case v1.TaintNodePIDPressure:
			messages[i] = "1 node has PID pressure"
			errCode = proto.StatusCode_STATUSCHECK_NODE_PID_PRESSURE
		case v1.TaintNodeNotReady:
			messages[i] = "1 node is not ready"
			if errCode == proto.StatusCode_STATUSCHECK_UNKNOWN_UNSCHEDULABLE {
				errCode = proto.StatusCode_STATUSCHECK_NODE_NOT_READY
			}
		case v1.TaintNodeUnreachable:
			messages[i] = "1 node is unreachable"
			if errCode == proto.StatusCode_STATUSCHECK_UNKNOWN_UNSCHEDULABLE {
				errCode = proto.StatusCode_STATUSCHECK_NODE_UNREACHABLE
			}
		case v1.TaintNodeUnschedulable:
			messages[i] = "1 node is unschedulable"
			if errCode == proto.StatusCode_STATUSCHECK_UNKNOWN_UNSCHEDULABLE {
				errCode = proto.StatusCode_STATUSCHECK_NODE_UNSCHEDULABLE
			}
		case v1.TaintNodeNetworkUnavailable:
			messages[i] = "1 node's network not available"
			if errCode == proto.StatusCode_STATUSCHECK_UNKNOWN_UNSCHEDULABLE {
				errCode = proto.StatusCode_STATUSCHECK_NODE_NETWORK_UNAVAILABLE
			}
		}
	}
	return errCode, fmt.Errorf("%s: 0/%d nodes available: %s", reason, len(messages), strings.Join(messages, ", "))
}

type podStatus struct {
	name       string
	namespace  string
	phase      string
	err        error
	statusCode proto.StatusCode
}

func (p *podStatus) isStable() bool {
	return p.phase == success || (p.phase == running && p.err == nil)
}

func (p *podStatus) withErr(errCode proto.StatusCode, err error) *podStatus {
	p.err = err
	p.statusCode = errCode
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

func extractErrorMessageFromWaitingContainerStatus(c v1.ContainerStatus) (proto.StatusCode, error) {
	switch c.State.Waiting.Reason {
	// Extract meaning full error out of container statuses.
	case containerCreating:
		return proto.StatusCode_STATUSCHECK_CONTAINER_CREATING, fmt.Errorf("creating container %s", c.Name)
	case crashLoopBackOff:
		// TODO, in case of container restarting, return the original failure reason due to which container failed.
		// TODO Add pod logs
		return proto.StatusCode_STATUSCHECK_CONTAINER_RESTARTING, fmt.Errorf("container %s is backing off waiting to restart", c.Name)
	case imagePullErr, imagePullBackOff, errImagePullBackOff:
		return proto.StatusCode_STATUSCHECK_IMAGE_PULL_ERR, fmt.Errorf("container %s is waiting to start: %s can't be pulled", c.Name, c.Image)
	case runContainerError:
		match := runContainerRe.FindStringSubmatch(c.State.Waiting.Message)
		if len(match) != 0 {
			return proto.StatusCode_STATUSCHECK_RUN_CONTAINER_ERR, fmt.Errorf("container %s in error: %s", c.Name, trimSpace(match[3]))
		}
	}
	return proto.StatusCode_STATUSCHECK_CONTAINER_WAITING_UNKNOWN, fmt.Errorf("container %s in error: %s", c.Name, trimSpace(c.State.Waiting.Message))
}

func newPodStatus(n string, ns string, p string) *podStatus {
	return &podStatus{
		name:       n,
		namespace:  ns,
		phase:      p,
		statusCode: proto.StatusCode_STATUSCHECK_SUCCESS,
	}
}

func trimSpace(msg string) string {
	return strings.Trim(msg, " ")
}
