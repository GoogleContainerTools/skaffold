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
	"sort"
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
	kubernetesMaxDeadline = 600
	tabHeader             = " -"
	tab                   = "  "
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

type PodStatuses struct {
	pods map[string]*resource.Pod
}

func (p *PodStatuses) BuildOrUpdatePods(podResources []validator.Resource) bool {
	allDone := true
	p.pods = make(map[string]*resource.Pod)
	for _, pr := range podResources {
		pod, ok := p.pods[pr.Name()]
		if !ok {
			pod = resource.NewPod(pr.Name(), pr.Namespace())
		}
		allDone = allDone && pr.Error() == nil
		pod.UpdateStatus(string(pr.Status()), pr.Error())
		p.pods[pr.Name()] = pod
	}
	return allDone
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := pkgkubernetes.Client()
	event.StatusCheckEventStarted()
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}
	diagnostics := diag.New(runCtx.Namespaces).
		WithLabel(defaultLabeller.RunIDKeyValueString()).
		WithValidators([]validator.Validator{validator.NewPodValidator(client)})

	deployments, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller,
		getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds))

	deadline := statusCheckMaxDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds, deployments)

	if err != nil {
		return fmt.Errorf("could not fetch deployments: %w", err)
	}

	var wg sync.WaitGroup

	rc := newResourceCounter(len(deployments))

	// fetch all pods.
	//TODO this map needs to be synchronized - we are writing it concurrently from pollPodStatus while reading it in
	// printResourceStatus
	podsMap := &PodStatuses{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		pollPodStatus(ctx, diagnostics, podsMap, deadline)
	}()

	for _, d := range deployments {
		wg.Add(1)
		go func(r Resource) {
			defer wg.Done()
			// keep updating the resource status until it fails/succeeds/times out
			pollResourceStatus(ctx, runCtx, r)
			rc.markProcessed(r.Status().Error())
		}(d)
	}

	// Retrieve pending resource states
	go func() {
		printResourceStatus(ctx, out, deployments, deadline, podsMap, rc)
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
		if d.Spec.ProgressDeadlineSeconds == nil || *d.Spec.ProgressDeadlineSeconds == kubernetesMaxDeadline {
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
	logrus.Debugf("starting  pollResourceStatus %s", r)
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

// printStatusCheckResult provides the final, e.g.:
// - deplyoment/leeroy-web is ready. [1/2 deployment(s) still pending]
func printStatusCheckResult(out *strings.Builder, r Resource, rc resourceCounter) {
	err := r.Status().Error()
	if errors.Is(err, context.Canceled) {
		// Don't print the status summary if the user ctrl-Cd
		return
	}

	status := fmt.Sprintf("%s %s", tabHeader, r)
	if err != nil {
		event.ResourceStatusCheckEventFailed(r.String(), err)
		status = fmt.Sprintf("%s failed.%s Error: %s.\n",
			status,
			trimNewLine(getPendingMessage(rc.deployments.pending, rc.deployments.total)),
			trimNewLine(err.Error()),
		)
	} else {
		event.ResourceStatusCheckEventSucceeded(r.String())
		status = fmt.Sprintf("%s is ready.\n", status)
	}

	out.WriteString(status)
}

// Print resource statuses until all status check are completed or context is cancelled.
func printResourceStatus(ctx context.Context, out io.Writer, resources []Resource, deadline time.Duration, podMap *PodStatuses, rc *resourceCounter) {
	timeoutContext, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	for {
		var allResourcesCheckComplete bool
		select {
		case <-timeoutContext.Done():
			return
		case <-time.After(reportStatusTime):
			allResourcesCheckComplete = printStatus(resources, out, podMap, rc)
		}
		if allResourcesCheckComplete {
			return
		}
	}
}

func printStatus(resources []Resource, out io.Writer, pods *PodStatuses, rc *resourceCounter) bool {
	allResourcesCheckComplete := true
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Summary: %s\n", getPendingMessage(rc.deployments.pending, rc.deployments.total)))
	for _, r := range resources {
		if r.IsStatusCheckComplete() {
			printStatusCheckResult(&result, r, *rc)
			continue
		}
		allResourcesCheckComplete = false
		headerWritten := false
		if status := r.ReportSinceLastUpdated(); status != "" {
			result.WriteString(fmt.Sprintf("%s %s\n", tabHeader, trimNewLine(status)))
			headerWritten = true
			event.ResourceStatusCheckEventUpdated(r.String(), status)
		}
		// Print pending pod statuses for this resource if any.
		//TODO this should return a string
		pods.printPodStatus(r, headerWritten, &result)
	}
	fmt.Fprint(out, result.String())
	return allResourcesCheckComplete
}

func (p *PodStatuses) printPodStatus(r Resource, headerWritten bool, result *strings.Builder) {
	podMap := p.pods
	keys := make([]string, 0, len(podMap))
	for k := range podMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		p := podMap[k]
		if strings.HasPrefix(p.Name(), r.Name()) {
			if str := p.ReportSinceLastUpdated(); str != "" {
				if !headerWritten {
					result.WriteString(fmt.Sprintf("%s %s\n", tabHeader, trimNewLine(fmt.Sprintf("%s: %s\n", r, r.Status()))))
					headerWritten = true
				}
				result.WriteString(fmt.Sprintf("%s %s %s\n", tab, tabHeader, str))
			}
		}
	}
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

func (c *resourceCounter) isDone() bool {
	return c.deployments.pending == 0 && c.pods.pending == 0
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

func pollPodStatus(ctx context.Context, d diag.Diagnose, podMap *PodStatuses, deadline time.Duration) error {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, deadline+pollDuration)
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			return nil
		case <-time.After(pollDuration):
			if pods, err := d.Run(); err == nil {
				if allDone := podMap.BuildOrUpdatePods(pods); allDone {
					return nil
				}
			}
		}
	}

}
