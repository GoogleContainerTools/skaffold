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
	"os/exec"
	"regexp"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/diag/recommender"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
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
	podInitializing     = "PodInitializing"
	podKind             = "pod"

	failedScheduling = "FailedScheduling"
	unhealthy        = "Unhealthy"
	execFmtError     = "exec format error"
)

var (
	runContainerRe = regexp.MustCompile(errorPrefix)
	taintsRe       = regexp.MustCompile(taintsExp)
	// for testing
	runCli        = executeCLI
	getReplicaSet = deploymentutil.GetAllReplicaSets

	unknownConditionsOrSuccess = map[proto.StatusCode]struct{}{
		proto.StatusCode_STATUSCHECK_UNKNOWN:                   {},
		proto.StatusCode_STATUSCHECK_CONTAINER_WAITING_UNKNOWN: {},
		proto.StatusCode_STATUSCHECK_UNKNOWN_UNSCHEDULABLE:     {},
		proto.StatusCode_STATUSCHECK_SUCCESS:                   {},
		proto.StatusCode_STATUSCHECK_POD_INITIALIZING:          {},
	}
)

// PodValidator implements the Validator interface for Pods
type PodValidator struct {
	k           kubernetes.Interface
	podSelector PodSelector
	recos       []Recommender
}

// NewPodValidator initializes a PodValidator
func NewPodValidator(k kubernetes.Interface, s PodSelector) *PodValidator {
	rs := []Recommender{recommender.ContainerError{}}
	return &PodValidator{k: k, recos: rs, podSelector: s}
}

// Validate implements the Validate method for Validator interface
func (p *PodValidator) Validate(ctx context.Context, ns string, opts metav1.ListOptions) ([]Resource, error) {
	pods, err := p.podSelector.Select(ctx, ns, opts)
	if err != nil {
		return []Resource{}, err
	}
	eventsClient := p.k.CoreV1().Events(ns)
	var rs []Resource
	for _, po := range pods {
		ps := p.getPodStatus(&po)
		// Update Pod status from Pod events if required
		updated := processPodEvents(eventsClient, po, ps)
		// The GVK group is not populated for List Objects. Hence set `kind` to `pod`
		// See https://github.com/kubernetes-sigs/controller-runtime/pull/389
		if po.Kind == "" {
			po.Kind = podKind
		}
		// Add recommendations
		for _, r := range p.recos {
			if s := r.Make(updated.ae.ErrCode); s.SuggestionCode != proto.SuggestionCode_NIL {
				updated.ae.Suggestions = append(updated.ae.Suggestions, s)
			}
		}
		rs = append(rs, NewResourceFromObject(&po, Status(updated.phase), &updated.ae, updated.logs))
	}
	return rs, nil
}

func (p *PodValidator) getPodStatus(pod *v1.Pod) *podStatus {
	ps := newPodStatus(pod.Name, pod.Namespace, string(pod.Status.Phase))
	switch pod.Status.Phase {
	case v1.PodSucceeded:
		return ps
	default:
		return ps.withErrAndLogs(getPodStatus(pod))
	}
}

func getPodStatus(pod *v1.Pod) (proto.StatusCode, []string, error) {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions

	// If the event type PodReady with status True is found then we return success immediately
	if isPodReady(pod) {
		return proto.StatusCode_STATUSCHECK_SUCCESS, nil, nil
	}
	// If the event type PodScheduled with status False is found then we check if it is due to taints and tolerations.
	if c, ok := isPodNotScheduled(pod); ok {
		log.Entry(context.TODO()).Debugf("Pod %q not scheduled: checking tolerations", pod.Name)
		sc, err := getUntoleratedTaints(c.Reason, c.Message)
		return sc, nil, err
	}
	// we can check the container status if the pod has been scheduled successfully. This can be determined by having the event
	// PodScheduled with status True, or a ContainerReady or PodReady event with status False.
	if isPodScheduledButNotReady(pod) {
		log.Entry(context.TODO()).Debugf("Pod %q scheduled but not ready: checking container statuses", pod.Name)
		// TODO(dgageot): Add EphemeralContainerStatuses
		cs := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
		// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states
		statusCode, logs, err := getContainerStatus(pod, cs)
		if statusCode == proto.StatusCode_STATUSCHECK_POD_INITIALIZING {
			// Determine if an init container is still running and fetch the init logs.
			for _, c := range pod.Status.InitContainerStatuses {
				if c.State.Waiting != nil {
					return statusCode, []string{}, fmt.Errorf("waiting for init container %s to start", c.Name)
				} else if c.State.Running != nil {
					sc, l := getPodLogs(pod, c.Name, statusCode)
					return sc, l, fmt.Errorf("waiting for init container %s to complete", c.Name)
				}
			}
		}
		return statusCode, logs, err
	}

	if c, ok := isPodStatusUnknown(pod); ok {
		log.Entry(context.TODO()).Debugf("Pod %q condition status of type %s is unknown", pod.Name, c.Type)
		return proto.StatusCode_STATUSCHECK_UNKNOWN, nil, fmt.Errorf(c.Message)
	}

	log.Entry(context.TODO()).Debugf("Unable to determine current service state of pod %q", pod.Name)
	return proto.StatusCode_STATUSCHECK_UNKNOWN, nil, fmt.Errorf("unable to determine current service state of pod %q", pod.Name)
}

func isPodReady(pod *v1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodReady && c.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func isPodNotScheduled(pod *v1.Pod) (v1.PodCondition, bool) {
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodScheduled && c.Status == v1.ConditionFalse {
			return c, true
		}
	}
	return v1.PodCondition{}, false
}

func isPodScheduledButNotReady(pod *v1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodScheduled && c.Status == v1.ConditionTrue {
			return true
		}
		if c.Type == v1.ContainersReady && c.Status == v1.ConditionFalse {
			return true
		}
		if c.Type == v1.PodReady && c.Status == v1.ConditionFalse {
			return true
		}
	}
	return false
}

func isPodStatusUnknown(pod *v1.Pod) (v1.PodCondition, bool) {
	for _, c := range pod.Status.Conditions {
		if c.Status == v1.ConditionUnknown {
			return c, true
		}
	}
	return v1.PodCondition{}, false
}

func getContainerStatus(po *v1.Pod, cs []v1.ContainerStatus) (proto.StatusCode, []string, error) {
	// See https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states
	for _, c := range cs {
		switch {
		case c.State.Waiting != nil:
			return extractErrorMessageFromWaitingContainerStatus(po, c)
		case c.State.Terminated != nil && c.State.Terminated.ExitCode != 0:
			sc, l := getPodLogs(po, c.Name, proto.StatusCode_STATUSCHECK_CONTAINER_TERMINATED)
			return sc, l, fmt.Errorf("container %s terminated with exit code %d", c.Name, c.State.Terminated.ExitCode)
		}
	}
	// No waiting or terminated containers, pod should be in good health.
	return proto.StatusCode_STATUSCHECK_SUCCESS, nil, nil
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

func processPodEvents(e corev1.EventInterface, pod v1.Pod, ps *podStatus) *podStatus {
	updated := ps
	if _, ok := unknownConditionsOrSuccess[ps.ae.ErrCode]; !ok {
		return updated
	}
	log.Entry(context.TODO()).Debugf("Fetching events for pod %q", pod.Name)
	// Get pod events.
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &pod)
	events, err := e.Search(scheme, &pod)
	if err != nil {
		log.Entry(context.TODO()).Debugf("Could not fetch events for resource %q due to %v", pod.Name, err)
		return updated
	}
	// find the latest event.
	var recentEvent *v1.Event
	for _, e := range events.Items {
		event := e.DeepCopy()
		if recentEvent == nil || recentEvent.LastTimestamp.Before(&event.LastTimestamp) {
			recentEvent = event
		}
	}
	if recentEvent == nil || recentEvent.Type == v1.EventTypeNormal {
		return updated
	}
	switch recentEvent.Reason {
	case failedScheduling:
		updated.updateAE(proto.StatusCode_STATUSCHECK_FAILED_SCHEDULING, recentEvent.Message)
	case unhealthy:
		updated.updateAE(proto.StatusCode_STATUSCHECK_UNHEALTHY, recentEvent.Message)
	default:
		// TODO: Add unique error codes for reasons
		updated.updateAE(
			proto.StatusCode_STATUSCHECK_UNKNOWN_EVENT,
			fmt.Sprintf("%s: %s", recentEvent.Reason, recentEvent.Message),
		)
	}

	return updated
}

type podStatus struct {
	name      string
	namespace string
	phase     string
	logs      []string
	ae        proto.ActionableErr
}

func (p *podStatus) isStable() bool {
	return p.phase == success || (p.phase == running && p.ae.Message == "")
}

func (p *podStatus) withErrAndLogs(errCode proto.StatusCode, l []string, err error) *podStatus {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	p.updateAE(errCode, msg)
	p.logs = l
	return p
}

func (p *podStatus) updateAE(errCode proto.StatusCode, msg string) {
	p.ae.ErrCode = errCode
	p.ae.Message = msg
}

func (p *podStatus) String() string {
	switch {
	case p.isStable():
		return ""
	default:
		if p.ae.Message != "" {
			return p.ae.Message
		}
	}
	return fmt.Sprintf(actionableMessage, p.namespace, p.name)
}

func extractErrorMessageFromWaitingContainerStatus(po *v1.Pod, c v1.ContainerStatus) (proto.StatusCode, []string, error) {
	// Extract meaning full error out of container statuses.
	switch c.State.Waiting.Reason {
	case podInitializing:
		// container is waiting to run. This could be because one of the init containers is
		// still not completed
		return proto.StatusCode_STATUSCHECK_POD_INITIALIZING, nil, nil
	case containerCreating:
		return proto.StatusCode_STATUSCHECK_CONTAINER_CREATING, nil, fmt.Errorf("creating container %s", c.Name)
	case crashLoopBackOff:
		// TODO, in case of container restarting, return the original failure reason due to which container failed.
		sc, l := getPodLogs(po, c.Name, proto.StatusCode_STATUSCHECK_CONTAINER_RESTARTING)
		return sc, l, fmt.Errorf("container %s is backing off waiting to restart", c.Name)
	case imagePullErr, imagePullBackOff, errImagePullBackOff:
		return proto.StatusCode_STATUSCHECK_IMAGE_PULL_ERR, nil, fmt.Errorf("container %s is waiting to start: %s can't be pulled", c.Name, c.Image)
	case runContainerError:
		match := runContainerRe.FindStringSubmatch(c.State.Waiting.Message)
		if len(match) != 0 {
			return proto.StatusCode_STATUSCHECK_RUN_CONTAINER_ERR, nil, fmt.Errorf("container %s in error: %s", c.Name, trimSpace(match[3]))
		}
	}
	log.Entry(context.TODO()).Debugf("Unknown waiting reason for container %q: %v", c.Name, c.State)
	return proto.StatusCode_STATUSCHECK_CONTAINER_WAITING_UNKNOWN, nil, fmt.Errorf("container %s in error: %v", c.Name, c.State.Waiting)
}

func newPodStatus(n string, ns string, p string) *podStatus {
	return &podStatus{
		name:      n,
		namespace: ns,
		phase:     p,
		ae: proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
		},
	}
}

func trimSpace(msg string) string {
	return strings.Trim(msg, " ")
}

func getPodLogs(po *v1.Pod, c string, sc proto.StatusCode) (proto.StatusCode, []string) {
	log.Entry(context.TODO()).Debugf("Fetching logs for container %s/%s", po.Name, c)
	logCommand := []string{"kubectl", "logs", po.Name, "-n", po.Namespace, "-c", c}
	logs, err := runCli(logCommand[0], logCommand[1:])
	if err != nil {
		return sc, []string{fmt.Sprintf("Error retrieving logs for pod %s: %s.\nTry `%s`", po.Name, err, strings.Join(logCommand, " "))}
	}
	if strings.Contains(string(logs), execFmtError) {
		sc = proto.StatusCode_STATUSCHECK_CONTAINER_EXEC_ERROR
	}
	output := strings.Split(string(logs), "\n")
	// remove spurious empty lines (empty string or from trailing newline)
	lines := make([]string, 0, len(output))
	for _, s := range output {
		if s == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("[%s %s] %s", po.Name, c, s))
	}
	return sc, lines
}

func executeCLI(cmdName string, args []string) ([]byte, error) {
	cmd := exec.Command(cmdName, args...)
	return cmd.CombinedOutput()
}

func isPodOwnedBy(po v1.Pod, controller metav1.Object) bool {
	if controller == nil {
		return true
	}
	return metav1.IsControlledBy(&po, controller)
}
