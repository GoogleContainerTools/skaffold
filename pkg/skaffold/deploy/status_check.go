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

package deploy

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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resource"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/proto"
)

var (
	defaultStatusCheckDeadline = 2 * time.Minute

	// Poll period for checking set to 100 milliseconds
	defaultPollPeriodInMilliseconds = 100

	// report resource status for pending resources 0.5 second.
	reportStatusTime = 500 * time.Millisecond
)

const (
	tabHeader             = " -"
	kubernetesMaxDeadline = 600
)

var (
	deployContext map[string]string
)

type counter struct {
	total   int
	pending int32
	failed  int32
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	event.StatusCheckEventStarted()
	errCode, err := statusCheck(ctx, defaultLabeller, runCtx, out)
	event.StatusCheckEventEnded(errCode, err)
	return err
}

func statusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) (proto.StatusCode, error) {
	client, err := pkgkubernetes.Client()
	if err != nil {
		return proto.StatusCode_STATUSCHECK_KUBECTL_CLIENT_FETCH_ERR, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	deployContext = map[string]string{
		"clusterName": runCtx.KubeContext,
	}

	deployments := make([]*resource.Deployment, 0)
	for _, n := range runCtx.Namespaces {
		newDeployments, err := getDeployments(client, n, defaultLabeller,
			getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds))
		if err != nil {
			return proto.StatusCode_STATUSCHECK_DEPLOYMENT_FETCH_ERR, fmt.Errorf("could not fetch deployments: %w", err)
		}
		deployments = append(deployments, newDeployments...)
	}

	deadline := statusCheckMaxDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds, deployments)

	var wg sync.WaitGroup

	c := newCounter(len(deployments))

	for _, d := range deployments {
		wg.Add(1)
		go func(r *resource.Deployment) {
			defer wg.Done()
			// keep updating the resource status until it fails/succeeds/times out
			pollDeploymentStatus(ctx, runCtx, r)
			rcCopy := c.markProcessed(r.Status().Error())
			printStatusCheckSummary(out, r, rcCopy)
		}(d)
	}

	// Retrieve pending deployments statuses
	go func() {
		printDeploymentStatus(ctx, out, deployments, deadline)
	}()

	// Wait for all deployment statuses to be fetched
	wg.Wait()
	return getSkaffoldDeployStatus(c, deployments)
}

func getDeployments(client kubernetes.Interface, ns string, l *DefaultLabeller, deadlineDuration time.Duration) ([]*resource.Deployment, error) {
	deps, err := client.AppsV1().Deployments(ns).List(metav1.ListOptions{
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
			WithLabel(RunIDLabel, l.Labels()[RunIDLabel]).
			WithValidators([]validator.Validator{validator.NewPodValidator(client, deployContext)})

		for k, v := range d.Spec.Template.Labels {
			pd = pd.WithLabel(k, v)
		}

		deployments[i] = resource.NewDeployment(d.Name, d.Namespace, deadline).WithValidator(pd)
	}

	return deployments, nil
}

func pollDeploymentStatus(ctx context.Context, runCtx *runcontext.RunContext, r *resource.Deployment) {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, r.Deadline()+pollDuration)
	logrus.Debugf("checking status %s", r)
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			msg := fmt.Sprintf("could not stabilize within %v: %v", r.Deadline(), timeoutContext.Err())
			r.UpdateStatus(proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_DEADLINE_EXCEEDED,
				Message: msg})
			return
		case <-time.After(pollDuration):
			r.CheckStatus(timeoutContext, runCtx)
			if r.IsStatusCheckComplete() {
				return
			}
		}
	}
}

func getSkaffoldDeployStatus(c *counter, rs []*resource.Deployment) (proto.StatusCode, error) {
	if c.failed > 0 {
		err := fmt.Errorf("%d/%d deployment(s) failed", c.failed, c.total)
		for _, r := range rs {
			if r.StatusCode != proto.StatusCode_STATUSCHECK_SUCCESS {
				return r.FirstPodErrOccurred(), err
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

func printStatusCheckSummary(out io.Writer, r *resource.Deployment, c counter) {
	ae := r.Status().ActionableError()
	if ae.ErrCode == proto.StatusCode_STATUSCHECK_CONTEXT_CANCELLED {
		// Don't print the status summary if the user ctrl-C
		return
	}
	event.ResourceStatusCheckEventCompleted(r.String(), r.Status().ActionableError())
	status := fmt.Sprintf("%s %s", tabHeader, r)
	if ae.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
		status = fmt.Sprintf("%s failed.%s Error: %s.",
			status,
			trimNewLine(getPendingMessage(c.pending, c.total)),
			trimNewLine(ae.Message),
		)
	} else {
		status = fmt.Sprintf("%s is ready.%s", status, getPendingMessage(c.pending, c.total))
	}

	fmt.Fprintln(out, status)
}

// Print resource statuses until all status check are completed or context is cancelled.
func printDeploymentStatus(ctx context.Context, out io.Writer, deployments []*resource.Deployment, deadline time.Duration) {
	timeoutContext, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	for {
		var allDone bool
		select {
		case <-timeoutContext.Done():
			return
		case <-time.After(reportStatusTime):
			allDone = printStatus(deployments, out)
		}
		if allDone {
			return
		}
	}
}

func printStatus(deployments []*resource.Deployment, out io.Writer) bool {
	allDone := true
	for _, r := range deployments {
		if r.IsStatusCheckComplete() {
			continue
		}
		allDone = false
		if str := r.ReportSinceLastUpdated(); str != "" {
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
	if err != nil {
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

func statusCheckMaxDeadline(value int, deployments []*resource.Deployment) time.Duration {
	if value > 0 {
		return time.Duration(value) * time.Second
	}
	d := time.Duration(0)
	for _, r := range deployments {
		if r.Deadline() > d {
			d = r.Deadline()
		}
	}
	return d
}
