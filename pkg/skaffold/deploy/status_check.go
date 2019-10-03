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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resource"
	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

var (
	defaultStatusCheckDeadline = time.Duration(10) * time.Minute

	// Poll period for checking set to 100 milliseconds
	defaultPollPeriodInMilliseconds = 100

	// report resource status for pending resources 0.5 second.
	reportStatusTime = 500 * time.Millisecond
)

const (
	tabHeader = " -"
)

type counter struct {
	total   int
	pending int32
	failed  int32
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := pkgkubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting kubernetes client")
	}

	deadline := getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds)
	deployments, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller, deadline)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}

	wg := sync.WaitGroup{}

	c := newCounter(len(deployments))

	for _, d := range deployments {
		wg.Add(1)
		go func(r Resource) {
			defer wg.Done()
			pollResourceStatus(ctx, runCtx, r)
			pending := c.markProcessed(r.Status().Error())
			printStatusCheckSummary(out, r, pending, c.total)
		}(d)
	}

	// Retrieve pending resource states
	go func() {
		printResourceStatus(ctx, out, deployments, deadline)
	}()

	// Wait for all deployment status to be fetched
	wg.Wait()
	return getSkaffoldDeployStatus(c)
}

func getDeployments(client kubernetes.Interface, ns string, l *DefaultLabeller, deadlineDuration time.Duration) ([]Resource, error) {
	deps, err := client.AppsV1().Deployments(ns).List(metav1.ListOptions{
		LabelSelector: l.RunIDKeyValueString(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch deployments")
	}

	deployments := make([]Resource, 0, len(deps.Items))
	for _, d := range deps.Items {
		var deadline time.Duration
		if d.Spec.ProgressDeadlineSeconds == nil || *d.Spec.ProgressDeadlineSeconds > int32(deadlineDuration.Seconds()) {
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
			r.UpdateStatus(timeoutContext.Err().Error(), timeoutContext.Err())
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
	return fmt.Errorf("%d/%d deployment(s) failed", c.failed, c.total)
}

func getDeadline(d int) time.Duration {
	if d > 0 {
		return time.Duration(d) * time.Second
	}
	return defaultStatusCheckDeadline
}

func printStatusCheckSummary(out io.Writer, r Resource, pending int, total int) {
	status := fmt.Sprintf("%s %s", tabHeader, r)
	if err := r.Status().Error(); err != nil {
		status = fmt.Sprintf("%s failed.%s Error: %s.",
			status,
			trimNewLine(getPendingMessage(pending, total)),
			trimNewLine(err.Error()),
		)
	} else {
		status = fmt.Sprintf("%s is ready.%s", status, getPendingMessage(pending, total))
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
			color.Default.Fprintln(out, tabHeader, trimNewLine(str))
		}
	}
	return allResourcesCheckComplete
}

func getPendingMessage(pending int, total int) string {
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

func (c *counter) markProcessed(err error) int {
	if err != nil {
		atomic.AddInt32(&c.failed, 1)
	}
	return int(atomic.AddInt32(&c.pending, -1))
}
