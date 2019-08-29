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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resources"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	TimeOutErr = "TimeOutErr"
)

var (
	// defaultStatusCheckDeadline is set to 10 minutes
	defaultStatusCheckDeadline int32 = 600

	// Poll period for checking set to 100 milliseconds
	pollDuration = 100 * time.Millisecond

	// report resource status for pending resources every minute.
	reportStatusTime = 1 * time.Second
)

type Checker struct {
	context       context.Context
	runCtx        *runcontext.RunContext
	out           io.Writer
	labeller      *DefaultLabeller
	client        kubernetes.Interface
	numDeps       int
	processedDeps int32
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := kubernetesutil.GetClientset()
	if err != nil {
		return err
	}
	deadline := getDeadline(int32(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds))

	deployments, err := fetchDeployments(client, runCtx.Opts.Namespace, defaultLabeller, deadline)
	if err != nil {
		return err
	}

	rs := []Resource{}
	for _, r := range deployments {
		rs = append(rs, r)
	}

	checker := &Checker{
		context:  ctx,
		runCtx:   runCtx,
		labeller: defaultLabeller,
		client:   client,
		out:      out,
		numDeps:  len(deployments),
	}

	wg := &sync.WaitGroup{}

	for _, r := range rs {
		// Check resource status
		wg.Add(1)
		go func(checker *Checker, r Resource) {
			defer func(checker *Checker) {
				atomic.AddInt32(&checker.processedDeps, 1)
				checker.printStatusCheckSummary(r)
				wg.Done()
			}(checker)
			checker.CheckResourceStatus(r)
		}(checker, r)
	}

	// Retrieve pending resource states
	go func() {
		checker.printResourceStatus(rs, deadline)
	}()

	// Wait for all resource status to be fetched
	wg.Wait()
	if checker.isSkaffoldDeployInError(rs) {
		return fmt.Errorf("one or more deployed resources were in error")
	}
	return nil
}

func fetchDeployments(c kubernetes.Interface, ns string, labeller *DefaultLabeller, deadline int32) ([]*resources.Deployment, error) {
	deps, err := c.AppsV1().Deployments(ns).List(metav1.ListOptions{
		LabelSelector: labeller.RunIDKeyValueString(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch deployments")
	}

	deployments := make([]*resources.Deployment, len(deps.Items))

	for i, d := range deps.Items {
		var depDeadline int32
		if d.Spec.ProgressDeadlineSeconds == nil || *d.Spec.ProgressDeadlineSeconds > deadline {
			logrus.Debugf("no or higher progressDeadlineSeconds config found for deployment %s. Setting deadline to %d seconds", d.Name, defaultStatusCheckDeadline)
			depDeadline = deadline
		} else {
			depDeadline = *d.Spec.ProgressDeadlineSeconds
		}
		deployments[i] = resources.NewDeployment(d.Name, d.Namespace, time.Duration(depDeadline)*time.Second)
	}
	return deployments, nil
}

func (c *Checker) CheckResourceStatus(r Resource) {
	timeoutContext, cancel := context.WithTimeout(c.context, r.Deadline())
	logrus.Debugf("checking status for %s in namespace %s for %s", r.String(), r.Namespace(), r.Deadline())
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			r.UpdateStatus("", TimeOutErr, errors.Wrap(timeoutContext.Err(), fmt.Sprintf("resource %s could not be fetched within %v", r.String(), r.Deadline())))
			return
		case <-time.After(pollDuration):
			r.CheckStatus(timeoutContext, c.runCtx)
			if r.IsStatusCheckComplete() {
				return
			}
		}
	}
}

// Print resource statuses until all status check are completed or context is cancelled.
func (c *Checker) printResourceStatus(rs []Resource, deadline int32) {
	timeoutContext, cancel := context.WithTimeout(c.context, time.Duration(deadline)*time.Second)
	defer cancel()
	for {
		allResourcesCheckComplete := true
		select {
		case <-timeoutContext.Done():
			return
		case <-time.After(reportStatusTime):
			for _, r := range rs {
				if !r.IsStatusCheckComplete() {
					allResourcesCheckComplete = false
					r.ReportSinceLastUpdated(c.out)
				}
			}
		}
		if allResourcesCheckComplete {
			return
		}
	}
}

func (c *Checker) isSkaffoldDeployInError(rs []Resource) bool {
	for _, r := range rs {
		if r.Status().Error() != nil {
			return true
		}
	}
	return false
}

func (c *Checker) printStatusCheckSummary(r Resource) {
	waitingMsg := ""
	stats := make([]string, 0, 2)
	if numLeft := c.numDeps - int(c.processedDeps); numLeft > 0 {
		stats = append(stats, fmt.Sprintf("%d/%d deployment(s)", numLeft, c.numDeps))
	}
	if len(stats) > 0 {
		waitingMsg = fmt.Sprintf("[%s still pending]", strings.Join(stats, ", "))
	}
	if err := r.Status().Error(); err != nil {
		color.Default.Fprintln(c.out,
			fmt.Sprintf("%s failed %s. Error: %s.",
				r.String(),
				waitingMsg,
				strings.TrimSuffix(err.Error(), "\n"),
			),
		)
	} else {
		color.Default.Fprintln(c.out, fmt.Sprintf("%s %s is ready. %s", resources.TabHeader, r.String(), waitingMsg))
	}
}

func getDeadline(d int32) int32 {
	if d > 0 {
		return d
	}
	return defaultStatusCheckDeadline
}
