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

package resource

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/diag"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/diag/validator"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	protoV2 "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
)

const (
	deploymentRolloutSuccess   = "successfully rolled out"
	connectionErrMsg           = "Unable to connect to the server"
	killedErrMsg               = "signal: killed"
	clientSideThrottleErrMsg   = "due to client-side throttling"
	couldNotFindResourceErrMsg = "the server could not find the requested resource"
	defaultPodCheckDeadline    = 30 * time.Second
	tabHeader                  = " -"
	tab                        = "  "
	maxLogLines                = 3
)

// Type represents a kubernetes resource type to health check.
type Type string

var (
	statefulsetRolloutSuccess = regexp.MustCompile("(roll out|rolling update) complete")

	msgKubectlKilled            = "kubectl rollout status command interrupted\n"
	MsgKubectlConnection        = "kubectl connection error\n"
	msgStrategyTypeNotSupported = "rollout status is only available for RollingUpdate strategy type"

	nonRetryContainerErrors = map[proto.StatusCode]struct{}{
		proto.StatusCode_STATUSCHECK_IMAGE_PULL_ERR:       {},
		proto.StatusCode_STATUSCHECK_RUN_CONTAINER_ERR:    {},
		proto.StatusCode_STATUSCHECK_CONTAINER_TERMINATED: {},
		proto.StatusCode_STATUSCHECK_CONTAINER_RESTARTING: {},
	}

	ResourceTypes = struct {
		StandalonePods  Type
		Deployment      Type
		StatefulSet     Type
		ConfigConnector Type
	}{
		StandalonePods:  "standalone-pods",
		Deployment:      "deployment",
		StatefulSet:     "statefulset",
		ConfigConnector: "config-connector-resource",
	}
)

type Group map[string]*Resource

func (r Group) Add(d *Resource) {
	r[d.ID()] = d
}

func (r Group) Contains(d *Resource) bool {
	_, found := r[d.ID()]
	return found
}

func (r Group) Reset() {
	for k := range r {
		delete(r, k)
	}
}

type Resource struct {
	name             string
	namespace        string
	rType            Type
	status           Status
	statusCode       proto.StatusCode
	done             bool
	tolerateFailures bool
	deadline         time.Duration
	resources        map[string]validator.Resource
	resoureValidator diag.Diagnose
}

func (r *Resource) ID() string {
	return fmt.Sprintf("%s:%s:%s", r.name, r.namespace, r.rType)
}

func (r *Resource) Deadline() time.Duration {
	return r.deadline
}

func (r *Resource) UpdateStatus(ae *proto.ActionableErr) {
	updated := newStatus(ae)
	if r.status.Equal(updated) {
		r.status.changed = false
		return
	}
	r.status = updated
	r.statusCode = updated.ActionableError().ErrCode
	r.status.changed = true
	if ae.ErrCode == proto.StatusCode_STATUSCHECK_SUCCESS || isErrAndNotRetryAble(ae.ErrCode) {
		r.done = true
	}
}

func NewResource(name string, rType Type, ns string, deadline time.Duration, tolerateFailures bool) *Resource {
	return &Resource{
		name:             name,
		namespace:        ns,
		rType:            rType,
		status:           newStatus(&proto.ActionableErr{}),
		deadline:         deadline,
		resoureValidator: diag.New(nil),
		tolerateFailures: tolerateFailures,
	}
}

func (r *Resource) WithValidator(pd diag.Diagnose) *Resource {
	r.resoureValidator = pd
	return r
}

func (r *Resource) checkStandalonePodsStatus(ctx context.Context, cfg kubectl.Config) *proto.ActionableErr {
	if len(r.resources) == 0 {
		return &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_STANDALONE_PODS_PENDING}
	}
	kubeCtl := kubectl.NewCLI(cfg, "")
	var pendingPods []string
	for _, pod := range r.resources {
		switch pod.Status() {
		case "Failed":
			return &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN, Message: fmt.Sprintf("pod %s failed", pod.Name())}
		case "Running":
			b, _ := kubeCtl.RunOut(ctx, "get", "pod", pod.Name(), "-o", `jsonpath={..status.conditions[?(@.type=="Ready")].status}`, "--namespace", pod.Namespace())
			if ctx.Err() != nil {
				return &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_USER_CANCELLED}
			}
			if podReady, _ := strconv.ParseBool(string(b)); !podReady {
				pendingPods = append(pendingPods, pod.Name())
			}
		default:
			pendingPods = append(pendingPods, pod.Name())
		}
	}
	if len(pendingPods) > 0 {
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_STANDALONE_PODS_PENDING,
			Message: fmt.Sprintf("pods not ready: %v", pendingPods),
		}
	}
	return &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS}
}

func (r *Resource) checkConfigConnectorStatus() *proto.ActionableErr {
	if len(r.resources) == 0 {
		return &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS}
	}
	var pendingResources []string
	for _, resource := range r.resources {
		ae := resource.ActionableError()
		if ae == nil {
			continue
		}
		switch ae.ErrCode {
		case proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_FAILED, proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_TERMINATING:
			return ae
		case proto.StatusCode_STATUSCHECK_SUCCESS:
			continue
		default:
			pendingResources = append(pendingResources, resource.Name())
		}
	}
	if len(pendingResources) > 0 {
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS,
			Message: fmt.Sprintf("config connector resources not ready: %v", pendingResources),
		}
	}
	return &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS}
}

func (r *Resource) checkRolloutStatus(ctx context.Context, cfg kubectl.Config) *proto.ActionableErr {
	b, err := kubectl.NewCLI(cfg, "").RunOut(ctx, "rollout", "status", string(r.rType), r.name, "--namespace", r.namespace, "--watch=false")
	if ctx.Err() != nil {
		return &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_USER_CANCELLED}
	}
	details := r.cleanupStatus(string(b))
	if err != nil {
		return parseKubectlRolloutError(details, r.deadline, r.tolerateFailures, err)
	}

	client, cErr := kubernetesclient.Client(cfg.GetKubeContext())
	if cErr != nil {
		log.Entry(ctx).Debugf("error attempting to create kubernetes client for k8s event listing: %s", err)
	} else {
		err = checkK8sEventsForPodFailedCreateEvent(ctx, client, r.namespace, r.name)
	}

	// additional logic added here which checks kubernetes events to see if skaffold managed pod has a FailedCreatEvent
	// this can be raised by an admission controller and if we don't error here, skaffold will wait for the pod to come up
	// indefinitely even thought the admission controller has denied it
	return parseKubectlRolloutError(details, r.deadline, r.tolerateFailures, err)
}

func checkK8sEventsForPodFailedCreateEvent(ctx context.Context, client kubernetes.Interface, namespace string, deploymentName string) error {
	// Create a watcher for events
	eventList, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error attempting to list kubernetes events in namespace: %s, %w", namespace, err)
	}

	for _, event := range eventList.Items {
		if event.Reason == "FailedCreate" {
			if strings.HasPrefix(event.InvolvedObject.Name, deploymentName+"-") {
				errMsg := fmt.Sprintf("Failed to create Pod for Deployment %s: %s\n", deploymentName, event.Message)
				return fmt.Errorf(errMsg)
			}
		}
	}
	return nil
}

func (r *Resource) CheckStatus(ctx context.Context, cfg kubectl.Config) {
	var ae *proto.ActionableErr
	switch r.rType {
	case ResourceTypes.StandalonePods:
		ae = r.checkStandalonePodsStatus(ctx, cfg)
	case ResourceTypes.ConfigConnector:
		ae = r.checkConfigConnectorStatus()
	default:
		ae = r.checkRolloutStatus(ctx, cfg)
	}

	r.UpdateStatus(ae)
	// send event update in check status.
	// if deployment is successfully rolled out, send pod success event to make sure
	// all pod are marked as success in V2
	// See https://github.com/GoogleCloudPlatform/cloud-code-vscode-internal/issues/5277
	if ae.ErrCode == proto.StatusCode_STATUSCHECK_SUCCESS {
		for _, pod := range r.resources {
			eventV2.ResourceStatusCheckEventCompletedMessage(
				pod.String(),
				fmt.Sprintf("%s %s: running.\n", tabHeader, pod.String()),
				&protoV2.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS},
			)
		}
		return
	}
	if err := r.fetchPods(ctx); err != nil {
		log.Entry(ctx).Debugf("pod statuses could not be fetched this time due to %s", err)
	}
}

func (r *Resource) String() string {
	switch r.rType {
	case ResourceTypes.StandalonePods:
		return "pods"
	default:
		if r.namespace == "default" {
			return fmt.Sprintf("%s/%s", r.rType, r.name)
		}

		return fmt.Sprintf("%s:%s/%s", r.namespace, r.rType, r.name)
	}
}

func (r *Resource) Name() string {
	return r.name
}

func (r *Resource) Status() Status {
	return r.status
}

func (r *Resource) IsStatusCheckCompleteOrCancelled() bool {
	return r.done || r.statusCode == proto.StatusCode_STATUSCHECK_USER_CANCELLED
}

func (r *Resource) StatusMessage() string {
	for _, p := range r.resources {
		if s := p.ActionableError(); s.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
			return fmt.Sprintf("%s\n", s.Message)
		}
	}
	return r.status.String()
}

func (r *Resource) MarkComplete() {
	r.done = true
}

// ReportSinceLastUpdated returns a string representing rollout status along with tab header
// e.g.
//  - testNs:deployment/leeroy-app: waiting for rollout to complete. (1/2) pending
//      - testNs:pod/leeroy-app-xvbg : error pulling container image

func (r *Resource) ReportSinceLastUpdated(isMuted bool) string {
	if r.status.reported && !r.status.changed {
		return ""
	}
	r.status.reported = true
	if r.status.String() == "" {
		return ""
	}
	var result strings.Builder
	// Pod container statuses can be empty.
	// This can happen when
	// 1. No pods have been scheduled for the rollout
	// 2. All containers are in running phase with no errors.
	// In such case, avoid printing any status update for the rollout.
	for _, p := range r.resources {
		if s := p.ActionableError().Message; s != "" {
			result.WriteString(fmt.Sprintf("%s %s %s: %s\n", tab, tabHeader, p, s))
			// if logs are muted, write container logs to file and last 3 lines to
			// result.
			out, writeTrimLines, err := withLogFile(p.Name(), &result, p.Logs(), isMuted)
			if err != nil {
				log.Entry(context.TODO()).Debugf("could not create log file %v", err)
			}
			trimLines := []string{}
			for i, l := range p.Logs() {
				formattedLine := fmt.Sprintf("%s %s > %s\n", tab, tab, strings.TrimSuffix(l, "\n"))
				if isMuted && i >= len(p.Logs())-maxLogLines {
					trimLines = append(trimLines, formattedLine)
				}
				out.Write([]byte(formattedLine))
			}
			writeTrimLines(trimLines)
		}
	}
	return fmt.Sprintf("%s %s: %s%s", tabHeader, r, r.StatusMessage(), result.String())
}

func (r *Resource) cleanupStatus(msg string) string {
	switch r.rType {
	case ResourceTypes.Deployment:
		clean := strings.ReplaceAll(msg, `deployment "`+r.Name()+`" `, "")
		if len(clean) > 0 {
			clean = strings.ToLower(clean[0:1]) + clean[1:]
		}
		return clean
	default:
		return msg
	}
}

// parses out connection error
// $kubectl logs somePod -f
// Unable to connect to the server: dial tcp x.x.x.x:443: connect: network is unreachable

// Parses out errors when kubectl was killed on client side
// $kubectl logs testPod  -f
// 2020/06/18 17:28:31 service is running
// Killed: 9
func parseKubectlRolloutError(details string, deadline time.Duration, tolerateFailures bool, err error) *proto.ActionableErr {
	switch {
	// deployment rollouts have success messages like `deployment "skaffold-foo" successfully rolled out`
	case err == nil && strings.Contains(details, deploymentRolloutSuccess):
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
			Message: details,
		}
	// statefulset rollouts have success messages like `statefulset rolling update complete 2 pods at revision skaffold-foo`
	case err == nil && statefulsetRolloutSuccess.MatchString(details):
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
			Message: details,
		}
	case err == nil:
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING,
			Message: details,
		}
	case strings.Contains(err.Error(), killedErrMsg):
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_KUBECTL_PID_KILLED,
			Message: fmt.Sprintf("received Ctrl-C or deployments could not stabilize within %v: %s", deadline, msgKubectlKilled),
		}
	case tolerateFailures:
		log.Entry(context.TODO()).Debugf("kubectl rollout encountered error but deployment continuing "+
			"as skaffold is currently configured to tolerate failures, err: %s", err)
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING,
			Message: details,
		}
	case strings.Contains(err.Error(), clientSideThrottleErrMsg) ||
		strings.Contains(err.Error(), couldNotFindResourceErrMsg):
		log.Entry(context.TODO()).Debugf("kubectl rollout encountered error but deployment continuing "+
			"as it is likely a transient error, err: %s", err)
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING,
			Message: details,
		}
	case strings.Contains(err.Error(), connectionErrMsg):
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_KUBECTL_CONNECTION_ERR,
			Message: MsgKubectlConnection,
		}
	// statefulset rollouts that use OnDelete strategy type don't support monitoring rollout, treat it as
	// if the deployment just completed successfully
	case strings.Contains(err.Error(), msgStrategyTypeNotSupported):
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
			Message: details,
		}
	default:
		return &proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN,
			Message: err.Error(),
		}
	}
}

func isErrAndNotRetryAble(statusCode proto.StatusCode) bool {
	return statusCode != proto.StatusCode_STATUSCHECK_KUBECTL_CONNECTION_ERR &&
		statusCode != proto.StatusCode_STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING &&
		statusCode != proto.StatusCode_STATUSCHECK_STANDALONE_PODS_PENDING &&
		statusCode != proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS &&
		statusCode != proto.StatusCode_STATUSCHECK_NODE_UNSCHEDULABLE &&
		statusCode != proto.StatusCode_STATUSCHECK_UNKNOWN_UNSCHEDULABLE
}

// HasEncounteredUnrecoverableError goes through all pod statuses and return true
// if any cannot be recovered
func (r *Resource) HasEncounteredUnrecoverableError() bool {
	for _, p := range r.resources {
		if _, ok := nonRetryContainerErrors[p.ActionableError().ErrCode]; ok {
			return true
		}
	}
	return false
}

func (r *Resource) fetchPods(ctx context.Context) error {
	timeoutContext, cancel := context.WithTimeout(ctx, defaultPodCheckDeadline)
	defer cancel()
	pods, err := r.resoureValidator.Run(timeoutContext)
	if err != nil {
		return err
	}

	newResources := map[string]validator.Resource{}
	r.status.changed = false
	for _, p := range pods {
		originalPod, found := r.resources[p.String()]
		if !found || originalPod.StatusUpdated(p) {
			r.status.changed = true
			prefix := fmt.Sprintf("%s %s:", tabHeader, p.String())
			if p.ActionableError().ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS &&
				p.ActionableError().Message != "" {
				event.ResourceStatusCheckEventUpdated(p.String(), p.ActionableError())
				eventV2.ResourceStatusCheckEventUpdatedMessage(
					p.String(),
					prefix,
					sErrors.V2fromV1(p.ActionableError()))
			}
		}
		newResources[p.String()] = p
	}
	r.resources = newResources
	return nil
}

// StatusCode returns the rollout status code if the status check is cancelled
// or if no pod data exists for this rollout.
// If pods are fetched, this function returns the error code a pod container encountered.
func (r *Resource) StatusCode() proto.StatusCode {
	// do not process pod status codes
	// 1) the user aborted the run or
	// 2) if another rollout failed which cancelled this deployment status check
	// 3) the deployment is successful. In case of successful rollouts, the code doesn't fetch the updated pod statuses.
	if r.statusCode == proto.StatusCode_STATUSCHECK_USER_CANCELLED || r.statusCode == proto.StatusCode_STATUSCHECK_SUCCESS {
		return r.statusCode
	}
	for _, p := range r.resources {
		if s := p.ActionableError().ErrCode; s != proto.StatusCode_STATUSCHECK_SUCCESS {
			return s
		}
	}
	return r.statusCode
}

func (r *Resource) WithPodStatuses(scs []proto.StatusCode) *Resource {
	r.resources = map[string]validator.Resource{}
	for i, s := range scs {
		name := fmt.Sprintf("%s-%d", r.name, i)
		r.resources[name] = validator.NewResource("test", "pod", "foo", validator.Status("failed"),
			&proto.ActionableErr{Message: "pod failed", ErrCode: s}, nil)
	}
	return r
}
