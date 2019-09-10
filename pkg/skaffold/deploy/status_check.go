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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

var (
	defaultStatusCheckDeadline = time.Duration(10) * time.Minute

	// Poll period for checking set to 100 milliseconds
	defaultPollPeriodInMilliseconds = 100

	// For testing
	executeRolloutStatus = getRollOutStatus
)

const (
	tabHeader = " -"
)

type counter struct {
	total   int
	pending int32
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
	// Its safe to use sync.Map without locks here as each subroutine adds a different key to the map.
	syncMap := &sync.Map{}

	c := newCounter(len(deployments))

	for _, d := range deployments {
		wg.Add(1)
		go func(d *resource.Deployment) {
			defer wg.Done()
			err := pollDeploymentRolloutStatus(ctx, kubectl.NewFromRunContext(runCtx), d)
			syncMap.Store(d.String(), err)
			pending := c.markProcessed()
			printStatusCheckSummary(d, pending, c.total, err, out)
		}(d)
	}

	// Wait for all deployment status to be fetched
	wg.Wait()
	return getSkaffoldDeployStatus(syncMap)
}

func getDeployments(client kubernetes.Interface, ns string, l *DefaultLabeller, deadlineDuration time.Duration) ([]*resource.Deployment, error) {
	deps, err := client.AppsV1().Deployments(ns).List(metav1.ListOptions{
		LabelSelector: l.RunIDKeyValueString(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch deployments")
	}

	deployments := make([]*resource.Deployment, 0, len(deps.Items))
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

func pollDeploymentRolloutStatus(ctx context.Context, k *kubectl.CLI, d *resource.Deployment) error {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, d.Deadline()+pollDuration)
	logrus.Debugf("checking rollout status %s", d.String())
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			err := errors.Wrap(timeoutContext.Err(), fmt.Sprintf("deployment rollout status could not be fetched within %v", d.Deadline()))
			return err
		case <-time.After(pollDuration):
			status, err := executeRolloutStatus(timeoutContext, k, d.Name())
			if err != nil || strings.Contains(status, "successfully rolled out") {
				return err
			}
		}
	}
}

func getSkaffoldDeployStatus(m *sync.Map) error {
	var errorStrings []string
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

func printStatusCheckSummary(d *resource.Deployment, pending int, total int, err error, out io.Writer) {
	status := fmt.Sprintf("%s %s", tabHeader, d)
	if err != nil {
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

func (c *counter) markProcessed() int {
	return int(atomic.AddInt32(&c.pending, -1))
}
