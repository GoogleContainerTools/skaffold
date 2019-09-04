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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resources"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

var (
	defaultStatusCheckDeadline = time.Duration(10) * time.Minute

	// Poll period for checking set to 100 milliseconds
	defaultPollPeriodInMilliseconds = 100

	// report resource status for pending resources 0.5 second.
	reportStatusTime = 500 * time.Millisecond

	// For testing
	executeRolloutStatus = getRollOutStatus
)

type Checker struct {
	context       context.Context
	runCtx        *runcontext.RunContext
	out           io.Writer
	labeller      *DefaultLabeller
	client        *kubectl.CLI
	numDeps       int
	processedDeps int32
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := pkgkubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting kubernetes client")
	}

	deadline := getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds)

	rsMap, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller, deadline)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}

	wg := sync.WaitGroup{}
	// Its safe to use sync.Map without locks here as each subroutine adds a different key to the map.
	syncMap := &sync.Map{}
	kubeCtl := kubectl.NewFromRunContext(runCtx)

	checker := &Checker{
		context:  ctx,
		runCtx:   runCtx,
		labeller: defaultLabeller,
		client:   kubectl.NewFromRunContext(runCtx),
		out:      out,
		numDeps:  len(rsMap),
	}

	rs := make([]*resources.ResourceObj, 0, len(rsMap))
	for r, deadlineDuration := range rsMap {
		rs = append(rs, &r)
		wg.Add(1)
		go func(r *resources.ResourceObj, deadlineDuration time.Duration) {
			defer wg.Done()
			pollDeploymentRolloutStatus(ctx, kubeCtl, r, deadlineDuration, syncMap)
		}(&r, deadlineDuration)
	}

	// Retrieve pending resource states
	go func() {
		checker.printResourceStatus(rs, deadline)
	}()

	// Wait for all deployment status to be fetched
	wg.Wait()
	return getSkaffoldDeployStatus(syncMap)
}

func getDeployments(client kubernetes.Interface, ns string, l *DefaultLabeller, deadlineDuration time.Duration) (map[resources.ResourceObj]time.Duration, error) {
	deps, err := client.AppsV1().Deployments(ns).List(metav1.ListOptions{
		LabelSelector: l.RunIDKeyValueString(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch deployments")
	}

	depMap := map[resources.ResourceObj]time.Duration{}

	for _, d := range deps.Items {
		var deadline time.Duration
		if d.Spec.ProgressDeadlineSeconds == nil || *d.Spec.ProgressDeadlineSeconds > int32(deadlineDuration.Seconds()) {
			deadline = deadlineDuration
		} else {
			deadline = time.Duration(*d.Spec.ProgressDeadlineSeconds) * time.Second
		}
		r := resources.NewResource(d.Name, d.Namespace)
		depMap[*r] = deadline
	}

	return depMap, nil
}

func pollDeploymentRolloutStatus(ctx context.Context, k *kubectl.CLI, r *resources.ResourceObj, deadline time.Duration, syncMap *sync.Map) {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, deadline+pollDuration)
	logrus.Debugf("checking rollout status %s", r.Name())
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			syncMap.Store(r.Name(), errors.Wrap(timeoutContext.Err(), fmt.Sprintf("deployment rollout status could not be fetched within %v", deadline)))
			return
		case <-time.After(pollDuration):
			status, err := executeRolloutStatus(timeoutContext, k, r.Name())
			if err != nil {
				syncMap.Store(r.Name(), err)
				r.UpdateStatus("", err.Error(), err)
				r.MarkCheckComplete()
				return
			}
			if strings.Contains(status, "successfully rolled out") {
				syncMap.Store(r.Name(), nil)
				r.UpdateStatus(status, status, nil)
				r.MarkCheckComplete()
				return
			}
			r.UpdateStatus(status, status, nil)
		}
	}
}

func getSkaffoldDeployStatus(m *sync.Map) error {
	errorStrings := []string{}
	m.Range(func(k, v interface{}) bool {
		if t, ok := v.(error); ok {
			errorStrings = append(errorStrings, fmt.Sprintf("deployment %s failed due to %s", k, t.Error()))
		}
		return true
	})

	if len(errorStrings) == 0 {
		return nil
	}
	return fmt.Errorf("following deployments are not stable:\n%s", strings.Join(errorStrings, "\n"))
}

func getRollOutStatus(ctx context.Context, k *kubectl.CLI, dName string) (string, error) {
	b, err := k.RunOut(ctx, "rollout", "status", "deployment", dName, "--watch=false")
	return string(b), err
}

func getDeadline(d int) time.Duration {
	if d > 0 {
		return time.Duration(d) * time.Second
	}
	return defaultStatusCheckDeadline
}

// Print resource statuses until all status check are completed or context is cancelled.
func (c *Checker) printResourceStatus(rs []*resources.ResourceObj, deadline time.Duration) {
	timeoutContext, cancel := context.WithTimeout(c.context, deadline)
	defer cancel()
	for {
		var allResourcesCheckComplete bool
		select {
		case <-timeoutContext.Done():
			return
		case <-time.After(reportStatusTime):
			allResourcesCheckComplete = printStatus(rs, c.out)
		}
		if allResourcesCheckComplete {
			return
		}
	}
}

func printStatus(rs []*resources.ResourceObj, out io.Writer) bool {
	allResourcesCheckComplete := true
	for _, r := range rs {
		if !r.IsStatusCheckComplete() {
			allResourcesCheckComplete = false
			r.ReportSinceLastUpdated(out)
		}
	}
	return allResourcesCheckComplete
}
