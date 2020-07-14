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
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/diag"
	"github.com/GoogleContainerTools/skaffold/pkg/diag/validator"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/proto"
)

const (
	deploymentType          = "deployment"
	rollOutSuccess          = "successfully rolled out"
	connectionErrMsg        = "Unable to connect to the server"
	killedErrMsg            = "signal: killed"
	defaultPodCheckDeadline = 30 * time.Second
	tabHeader               = " -"
	tab                     = "  "
)

var (
	msgKubectlKilled     = "kubectl rollout status command interrupted"
	MsgKubectlConnection = "kubectl connection error"
)

type Deployment struct {
	name         string
	namespace    string
	rType        string
	status       Status
	StatusCode   proto.StatusCode
	done         bool
	deadline     time.Duration
	pods         map[string]validator.Resource
	podValidator diag.Diagnose
}

func (d *Deployment) Deadline() time.Duration {
	return d.deadline
}

func (d *Deployment) UpdateStatus(ae proto.ActionableErr) {
	updated := newStatus(ae)
	if d.status.Equal(updated) {
		d.status.changed = false
		return
	}
	d.status = updated
	d.status.changed = true
	if ae.ErrCode == proto.StatusCode_STATUSCHECK_SUCCESS || isErrAndNotRetryAble(ae.ErrCode) {
		d.done = true
	}
}

func NewDeployment(name string, ns string, deadline time.Duration) *Deployment {
	return &Deployment{
		name:         name,
		namespace:    ns,
		rType:        deploymentType,
		status:       newStatus(proto.ActionableErr{}),
		deadline:     deadline,
		podValidator: diag.New(nil),
	}
}

func (d *Deployment) WithValidator(pd diag.Diagnose) *Deployment {
	d.podValidator = pd
	return d
}

func (d *Deployment) CheckStatus(ctx context.Context, runCtx *runcontext.RunContext) {
	kubeCtl := kubectl.NewFromRunContext(runCtx)

	b, err := kubeCtl.RunOut(ctx, "rollout", "status", "deployment", d.name, "--namespace", d.namespace, "--watch=false")
	if ctx.Err() != nil {
		return
	}

	details := d.cleanupStatus(string(b))

	ae := parseKubectlRolloutError(details, err)
	if ae.ErrCode == proto.StatusCode_STATUSCHECK_KUBECTL_PID_KILLED {
		ae.Message = fmt.Sprintf("received Ctrl-C or deployments could not stabilize within %v: %v", d.deadline, err)
	}

	d.UpdateStatus(ae)
	if err := d.fetchPods(ctx); err != nil {
		logrus.Debugf("pod statuses could be fetched this time due to %s", err)
	}
}

func (d *Deployment) String() string {
	if d.namespace == "default" {
		return fmt.Sprintf("%s/%s", d.rType, d.name)
	}

	return fmt.Sprintf("%s:%s/%s", d.namespace, d.rType, d.name)
}

func (d *Deployment) Name() string {
	return d.name
}

func (d *Deployment) Status() Status {
	return d.status
}

func (d *Deployment) IsStatusCheckComplete() bool {
	return d.done
}

// This returns a string representing deployment status along with tab header
// e.g.
//  - testNs:deployment/leeroy-app: waiting for rollout to complete. (1/2) pending
//      - testNs:pod/leeroy-app-xvbg : error pulling container image
func (d *Deployment) ReportSinceLastUpdated() string {
	if d.status.reported && !d.status.changed {
		return ""
	}
	d.status.reported = true
	if d.status.String() == "" {
		return ""
	}
	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s %s: %s", tabHeader, d, d.status))
	for _, p := range d.pods {
		if s := p.ActionableError().Message; s != "" {
			result.WriteString(fmt.Sprintf("%s %s %s: %s\n", tab, tabHeader, p, s))
			for _, l := range p.Logs() {
				result.WriteString(fmt.Sprintf("%s\n", l))
			}
		}
	}
	return result.String()
}

func (d *Deployment) cleanupStatus(msg string) string {
	clean := strings.ReplaceAll(msg, `deployment "`+d.Name()+`" `, "")
	if len(clean) > 0 {
		clean = strings.ToLower(clean[0:1]) + clean[1:]
	}
	return clean
}

// parses out connection error
// $kubectl logs somePod -f
// Unable to connect to the server: dial tcp x.x.x.x:443: connect: network is unreachable

// Parses out errors when kubectl was killed on client side
// $kubectl logs testPod  -f
// 2020/06/18 17:28:31 service is running
// Killed: 9
func parseKubectlRolloutError(details string, err error) proto.ActionableErr {
	switch {
	case err == nil && strings.Contains(details, rollOutSuccess):
		return proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
			Message: details,
		}
	case err == nil:
		return proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING,
			Message: details,
		}
	case strings.Contains(err.Error(), connectionErrMsg):
		return proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_KUBECTL_CONNECTION_ERR,
			Message: MsgKubectlConnection,
		}
	case strings.Contains(err.Error(), killedErrMsg):
		return proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_KUBECTL_PID_KILLED,
			Message: msgKubectlKilled,
		}
	default:
		return proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN,
			Message: err.Error(),
		}
	}
}

func isErrAndNotRetryAble(statusCode proto.StatusCode) bool {
	return statusCode != proto.StatusCode_STATUSCHECK_KUBECTL_CONNECTION_ERR &&
		statusCode != proto.StatusCode_STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING
}

func (d *Deployment) fetchPods(ctx context.Context) error {
	timeoutContext, cancel := context.WithTimeout(ctx, defaultPodCheckDeadline)
	defer cancel()
	pods, err := d.podValidator.Run(timeoutContext)
	if err != nil {
		return err
	}

	newPods := map[string]validator.Resource{}
	d.status.changed = false
	for _, p := range pods {
		originalPod, found := d.pods[p.String()]
		if !found || originalPod.StatusUpdated(p) {
			d.status.changed = true
			switch p.ActionableError().ErrCode {
			case proto.StatusCode_STATUSCHECK_CONTAINER_CREATING:
				event.ResourceStatusCheckEventUpdated(p.String(), p.ActionableError())
			default:
				event.ResourceStatusCheckEventCompleted(p.String(), p.ActionableError())
			}
		}
		newPods[p.String()] = p
	}
	d.pods = newPods
	return nil
}

// Return first pod status in error.
// TODO: should we return all distinct error codes in future?
func (d *Deployment) FirstPodErrOccurred() proto.StatusCode {
	for _, p := range d.pods {
		if s := p.ActionableError().ErrCode; s != proto.StatusCode_STATUSCHECK_SUCCESS {
			return s
		}
	}
	return proto.StatusCode_STATUSCHECK_SUCCESS
}

func (d *Deployment) WithPodStatuses(scs []proto.StatusCode) *Deployment {
	d.pods = map[string]validator.Resource{}
	for i, s := range scs {
		name := fmt.Sprintf("%s-%d", d.name, i)
		d.pods[name] = validator.NewResource("test", "pod", "foo", validator.Status("failed"),
			proto.ActionableErr{Message: "pod failed", ErrCode: s}, nil)
	}
	return d
}
