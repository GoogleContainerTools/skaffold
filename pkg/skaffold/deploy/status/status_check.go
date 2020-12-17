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

package status

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/pkg/diag"
	"github.com/GoogleContainerTools/skaffold/pkg/diag/validator"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resource"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/proto"
)

var (
	defaultStatusCheckDeadline = 2 * time.Minute

	// Poll period for checking set to 1 second
	defaultPollPeriodInMilliseconds = 1000

	// report resource status for pending resources 5 seconds.
	reportStatusTime = 5 * time.Second
)

const (
	tabHeader             = " -"
	kubernetesMaxDeadline = 600
)

type counter struct {
	total   int
	pending int32
	failed  int32
}

type Config interface {
	kubectl.Config

	GetNamespaces() []string
	StatusCheckDeadlineSeconds() int
	Muted() config.Muted
}

// Checker waits for the application to be totally deployed.
type Checker interface {
	Check(context.Context, io.Writer) error
}

// statusChecker runs status checks for pods and deployments
type statusChecker struct {
	cfg             Config
	labeller        *label.DefaultLabeller
	deadlineSeconds int
	muteLogs        bool
}

// NewStatusChecker returns a status checker which runs checks on deployments and pods.
func NewStatusChecker(cfg Config, labeller *label.DefaultLabeller) Checker {
	return statusChecker{
		muteLogs:        cfg.Muted().MuteStatusCheck(),
		cfg:             cfg,
		labeller:        labeller,
		deadlineSeconds: cfg.StatusCheckDeadlineSeconds(),
	}
}

// Run runs the status checks on deployments and pods deployed in current skaffold dev iteration.
func (s statusChecker) Check(ctx context.Context, out io.Writer) error {
	event.StatusCheckEventStarted()
	errCode, err := s.statusCheck(ctx, out)
	event.StatusCheckEventEnded(errCode, err)
	return err
}

func (s statusChecker) statusCheck(ctx context.Context, out io.Writer) (proto.StatusCode, error) {
	client, err := kubernetesclient.Client()
	if err != nil {
		return proto.StatusCode_STATUSCHECK_KUBECTL_CLIENT_FETCH_ERR, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	deployments := make([]*resource.Deployment, 0)
	for _, n := range s.cfg.GetNamespaces() {
		newDeployments, err := getDeployments(ctx, client, n, s.labeller,
			getDeadline(s.deadlineSeconds))
		if err != nil {
			return proto.StatusCode_STATUSCHECK_DEPLOYMENT_FETCH_ERR, fmt.Errorf("could not fetch deployments: %w", err)
		}
		deployments = append(deployments, newDeployments...)
	}

	var wg sync.WaitGroup

	c := newCounter(len(deployments))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, d := range deployments {
		wg.Add(1)
		go func(r *resource.Deployment) {
			defer wg.Done()
			// keep updating the resource status until it fails/succeeds/times out
			pollDeploymentStatus(ctx, s.cfg, r)
			rcCopy := c.markProcessed(r.Status().Error())
			s.printStatusCheckSummary(out, r, rcCopy)
			// if one deployment fails, cancel status checks for all deployments.
			if r.Status().Error() != nil && r.StatusCode() != proto.StatusCode_STATUSCHECK_USER_CANCELLED {
				cancel()
			}
		}(d)
	}

	// Retrieve pending deployments statuses
	go func() {
		s.printDeploymentStatus(ctx, out, deployments)
	}()

	// Wait for all deployment statuses to be fetched
	wg.Wait()
	cancel()
	return getSkaffoldDeployStatus(c, deployments)
}

func getDeployments(ctx context.Context, client kubernetes.Interface, ns string, l *label.DefaultLabeller, deadlineDuration time.Duration) ([]*resource.Deployment, error) {
	deps, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{
		LabelSelector: l.RunIDSelector(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch deployments: %w", err)
	}

	deployments := make([]*resource.Deployment, len(deps.Items))
	for i, d := range deps.Items {
		var deadline time.Duration
		if d.Spec.ProgressDeadlineSeconds == nil || *d.Spec.ProgressDeadlineSeconds == kubernetesMaxDeadline {
			deadline = deadlineDuration
		} else {
			deadline = time.Duration(*d.Spec.ProgressDeadlineSeconds) * time.Second
		}
		pd := diag.New([]string{d.Namespace}).
			WithLabel(label.RunIDLabel, l.Labels()[label.RunIDLabel]).
			WithValidators([]validator.Validator{validator.NewPodValidator(client)})

		for k, v := range d.Spec.Template.Labels {
			pd = pd.WithLabel(k, v)
		}

		deployments[i] = resource.NewDeployment(d.Name, d.Namespace, deadline).WithValidator(pd)
	}
	return deployments, nil
}

func pollDeploymentStatus(ctx context.Context, cfg pkgkubectl.Config, r *resource.Deployment) {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, r.Deadline()+pollDuration)
	logrus.Debugf("checking status %s", r)
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			switch c := timeoutContext.Err(); c {
			case context.Canceled:
				r.UpdateStatus(proto.ActionableErr{
					ErrCode: proto.StatusCode_STATUSCHECK_USER_CANCELLED,
					Message: "check cancelled\n",
				})
			case context.DeadlineExceeded:
				r.UpdateStatus(proto.ActionableErr{
					ErrCode: proto.StatusCode_STATUSCHECK_DEADLINE_EXCEEDED,
					Message: fmt.Sprintf("could not stabilize within %v\n", r.Deadline()),
				})
			}
			return
		case <-time.After(pollDuration):
			r.CheckStatus(timeoutContext, cfg)
			if r.IsStatusCheckCompleteOrCancelled() {
				return
			}
			// Fail immediately if any pod container errors cannot be recovered.
			// StatusCheck is not interruptable.
			// As any changes to build or deploy dependencies are not triggered, exit
			// immediately rather than waiting for for statusCheckDeadlineSeconds
			// TODO: https://github.com/GoogleContainerTools/skaffold/pull/4591
			if r.HasEncounteredUnrecoverableError() {
				r.MarkComplete()
				return
			}
		}
	}
}

func getSkaffoldDeployStatus(c *counter, rs []*resource.Deployment) (proto.StatusCode, error) {
	if c.failed > 0 {
		err := fmt.Errorf("%d/%d deployment(s) failed", c.failed, c.total)
		for _, r := range rs {
			if r.StatusCode() != proto.StatusCode_STATUSCHECK_SUCCESS {
				return r.StatusCode(), err
			}
		}
	}
	return proto.StatusCode_STATUSCHECK_SUCCESS, nil
}

func getDeadline(d int) time.Duration {
	if d > 0 {
		return time.Duration(d) * time.Second
	}
	return defaultStatusCheckDeadline
}

func (s statusChecker) printStatusCheckSummary(out io.Writer, r *resource.Deployment, c counter) {
	ae := r.Status().ActionableError()
	if r.StatusCode() == proto.StatusCode_STATUSCHECK_USER_CANCELLED {
		// Don't print the status summary if the user ctrl-C or
		// another deployment failed
		return
	}
	event.ResourceStatusCheckEventCompleted(r.String(), ae)
	status := fmt.Sprintf("%s %s", tabHeader, r)
	if ae.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
		if str := r.ReportSinceLastUpdated(s.muteLogs); str != "" {
			fmt.Fprintln(out, trimNewLine(str))
		}
		status = fmt.Sprintf("%s failed. Error: %s.",
			status,
			trimNewLine(r.StatusMessage()),
		)
	} else {
		status = fmt.Sprintf("%s is ready.%s", status, getPendingMessage(c.pending, c.total))
	}

	fmt.Fprintln(out, status)
}

// printDeploymentStatus prints resource statuses until all status check are completed or context is cancelled.
func (s statusChecker) printDeploymentStatus(ctx context.Context, out io.Writer, deployments []*resource.Deployment) {
	for {
		var allDone bool
		select {
		case <-ctx.Done():
			return
		case <-time.After(reportStatusTime):
			allDone = s.printStatus(deployments, out)
		}
		if allDone {
			return
		}
	}
}

func (s statusChecker) printStatus(deployments []*resource.Deployment, out io.Writer) bool {
	allDone := true
	for _, r := range deployments {
		if r.IsStatusCheckCompleteOrCancelled() {
			continue
		}
		allDone = false
		if str := r.ReportSinceLastUpdated(s.muteLogs); str != "" {
			event.ResourceStatusCheckEventUpdated(r.String(), r.Status().ActionableError())
			fmt.Fprintln(out, trimNewLine(str))
		}
	}
	return allDone
}

func getPendingMessage(pending int32, total int) string {
	if pending > 0 {
		return fmt.Sprintf(" [%d/%d deployment(s) still pending]", pending, total)
	}
	return ""
}

func trimNewLine(msg string) string {
	return strings.TrimSuffix(msg, "\n")
}

func newCounter(i int) *counter {
	return &counter{
		total:   i,
		pending: int32(i),
	}
}

func (c *counter) markProcessed(err error) counter {
	if err != nil && err != context.Canceled {
		atomic.AddInt32(&c.failed, 1)
	}
	atomic.AddInt32(&c.pending, -1)
	return c.copy()
}

func (c *counter) copy() counter {
	return counter{
		total:   c.total,
		pending: c.pending,
		failed:  c.failed,
	}
}
