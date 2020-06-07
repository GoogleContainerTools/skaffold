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
	"errors"
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
	errKubectlKilled     = errors.New("kubectl rollout status command interrupted")
	ErrKubectlConnection = errors.New("kubectl connection error")
)

type Deployment struct {
	name         string
	namespace    string
	rType        string
	status       Status
	done         bool
	deadline     time.Duration
	pods         map[string]validator.Resource
	podValidator diag.Diagnose
}

func (d *Deployment) Deadline() time.Duration {
	return d.deadline
}

func (d *Deployment) UpdateStatus(details string, err error) {
	errCode := proto.StatusCode_STATUSCHECK_SUCCESS
	if err != nil {
		errCode = proto.StatusCode_STATUSCHECK_UNKNOWN
	}
	updated := newStatus(details, errCode, err)
	if d.status.Equal(updated) {
		d.status.changed = false
		return
	}
	d.status = updated
	d.status.changed = true
	if strings.Contains(details, rollOutSuccess) || isErrAndNotRetryAble(err) {
		d.done = true
	}
}

func NewDeployment(name string, ns string, deadline time.Duration) *Deployment {
	return &Deployment{
		name:         name,
		namespace:    ns,
		rType:        deploymentType,
		status:       newStatus("", proto.StatusCode_STATUSCHECK_UNKNOWN, nil),
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

	err = parseKubectlRolloutError(err)
	if err == errKubectlKilled {
		err = fmt.Errorf("received Ctrl-C or deployments could not stabilize within %v: %w", d.deadline, err)
	}

	d.UpdateStatus(details, err)
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
		if p.Error() != nil {
			result.WriteString(fmt.Sprintf("%s %s %s: %s\n", tab, tabHeader, p, p.Error()))
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

func parseKubectlRolloutError(err error) error {
	if err == nil {
		return err
	}
	if strings.Contains(err.Error(), connectionErrMsg) {
		return ErrKubectlConnection
	}
	if strings.Contains(err.Error(), killedErrMsg) {
		return errKubectlKilled
	}
	return err
}

func isErrAndNotRetryAble(err error) bool {
	if err == nil {
		return false
	}
	return err != ErrKubectlConnection
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
		originalPod, ok := d.pods[p.String()]
		if !ok {
			d.status.changed = true
			event.ResourceStatusCheckEventCompleted(p.String(), p.Error())
		} else if originalPod.StatusCode != p.StatusCode {
			d.status.changed = true
			event.ResourceStatusCheckEventCompleted(p.String(), p.Error())
		}
		newPods[p.String()] = p
	}
	d.pods = newPods
	return nil
}
