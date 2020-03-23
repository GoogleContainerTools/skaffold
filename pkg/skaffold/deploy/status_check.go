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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
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
	tabHeader = " -"
)

type resourceCounter struct {
	deployments *counter
	pods        *counter
}

type counter struct {
	total   int
	pending int32
	failed  int32
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := pkgkubernetes.Client()
	event.StatusCheckEventStarted()
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	deployments, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller,
		getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds))

	deadline := statusCheckMaxDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds, deployments)

	if err != nil {
		return fmt.Errorf("could not fetch deployments: %w", err)
	}

	var wg sync.WaitGroup

	rc := newResourceCounter(len(deployments))

	for _, d := range deployments {
		wg.Add(1)
		go func(r Resource) {
			defer wg.Done()
			pollResourceStatus(ctx, runCtx, r)
			rcCopy := rc.markProcessed(r.Status().Error())
			printStatusCheckSummary(out, r, rcCopy)
		}(d)
	}

	// Retrieve pending resource states
	go func() {
		printResourceStatus(ctx, out, deployments, deadline)
	}()

	// Wait for all deployment status to be fetched
	wg.Wait()
	return getSkaffoldDeployStatus(rc.deployments)
}

func getDeployments(client kubernetes.Interface, ns string, l *DefaultLabeller, deadlineDuration time.Duration) ([]Resource, error) {
	deps, err := client.AppsV1().Deployments(ns).List(metav1.ListOptions{
		LabelSelector: l.RunIDKeyValueString(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch deployments: %w", err)
	}

	deployments := make([]Resource, 0, len(deps.Items))
	for _, d := range deps.Items {
		var deadline time.Duration
		if d.Spec.ProgressDeadlineSeconds == nil {
			deadline = deadlineDuration
		} else {
			deadline = time.Duration(*d.Spec.ProgressDeadlineSeconds) * time.Second
		}
		deployments = append(deployments, resource.NewDeployment(d.Name, d.Namespace, deadline))
	}

	return deployments, nil
}

func pollResourceStatus(ctx context.Context, runCtx *runcontext.RunContext, r Resource) {
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
		event.StatusCheckEventSucceeded()
		return nil
	}
	err := fmt.Errorf("%d/%d deployment(s) failed", c.failed, c.total)
	event.StatusCheckEventFailed(err)
	return err
}

func getDeadline(d int) time.Duration {
	if d > 0 {
		return time.Duration(d) * time.Second
	}
	return defaultStatusCheckDeadline
}

func printStatusCheckSummary(out io.Writer, r Resource, rc resourceCounter) {
	status := fmt.Sprintf("%s %s", tabHeader, r)
	if err := r.Status().Error(); err != nil {
		event.ResourceStatusCheckEventFailed(r.String(), err)
		status = fmt.Sprintf("%s failed.%s Error: %s.",
			status,
			trimNewLine(getPendingMessage(rc.deployments.pending, rc.deployments.total)),
			trimNewLine(err.Error()),
		)
	} else {
		event.ResourceStatusCheckEventSucceeded(r.String())
		status = fmt.Sprintf("%s is ready.%s", status, getPendingMessage(rc.deployments.pending, rc.deployments.total))
	}
	color.Default.Fprintln(out, status)
}

// Print resource statuses until all status check are completed or context is cancelled.
func printResourceStatus(ctx context.Context, out io.Writer, resources []Resource, deadline time.Duration) {
	timeoutContext, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	for {
		var allResourcesCheckComplete bool
		select {
		case <-timeoutContext.Done():
			return
		case <-time.After(reportStatusTime):
			allResourcesCheckComplete = printStatus(resources, out)
		}
		if allResourcesCheckComplete {
			return
		}
	}
}

func printStatus(resources []Resource, out io.Writer) bool {
	allResourcesCheckComplete := true
	for _, r := range resources {
		if r.IsStatusCheckComplete() {
			continue
		}
		allResourcesCheckComplete = false
		if str := r.ReportSinceLastUpdated(); str != "" {
			event.ResourceStatusCheckEventUpdated(r.String(), str)
			color.Default.Fprintln(out, tabHeader, trimNewLine(str))
		}
	}
	return allResourcesCheckComplete
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

func newResourceCounter(d int) *resourceCounter {
	return &resourceCounter{
		deployments: newCounter(d),
		pods:        newCounter(0),
	}
}

func (c *resourceCounter) markProcessed(err error) resourceCounter {
	depCp := c.deployments.markProcessed(err)
	podCp := c.pods.copy()
	return resourceCounter{
		deployments: &depCp,
		pods:        &podCp,
	}
}

func statusCheckMaxDeadline(value int, deployments []Resource) time.Duration {
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
