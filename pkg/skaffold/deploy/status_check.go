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
	"errors"
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

type counter struct {
	total   int
	pending int32
	failed  int32
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	event.StatusCheckEventStarted()
	err := statusCheck(ctx, defaultLabeller, runCtx, out)
	event.StatusCheckEventEnded(err)
	return err
}

func statusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := pkgkubernetes.Client()
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	deployments, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller,
		getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds))
	if err != nil {
		return fmt.Errorf("could not fetch deployments: %w", err)
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
	return getSkaffoldDeployStatus(c)
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
			WithValidators([]validator.Validator{validator.NewPodValidator(client)})

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
			err := fmt.Errorf("could not stabilize within %v: %w", r.Deadline(), timeoutContext.Err())
			r.UpdateStatus(err.Error(), err)
			return
		case <-time.After(pollDuration):
			r.CheckStatus(timeoutContext, runCtx)
			if r.IsStatusCheckComplete() {
				return
			}
		}
	}
}

func getSkaffoldDeployStatus(c *counter) error {
	if c.failed == 0 {
		return nil
	}
	err := fmt.Errorf("%d/%d deployment(s) failed", c.failed, c.total)

	return err
}

func getDeadline(d int) time.Duration {
	if d > 0 {
		return time.Duration(d) * time.Second
	}
	return defaultStatusCheckDeadline
}

func printStatusCheckSummary(out io.Writer, r *resource.Deployment, c counter) {
	err := r.Status().Error()
	if errors.Is(err, context.Canceled) {
		// Don't print the status summary if the user ctrl-C
		return
	}
	event.ResourceStatusCheckEventCompleted(r.String(), err)
	status := fmt.Sprintf("%s %s", tabHeader, r)
	if err != nil {
		status = fmt.Sprintf("%s failed.%s Error: %s.",
			status,
			trimNewLine(getPendingMessage(c.pending, c.total)),
			trimNewLine(err.Error()),
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
			event.ResourceStatusCheckEventUpdated(r.String(), str)
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
