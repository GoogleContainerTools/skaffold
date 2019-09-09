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

	// For testing
	executeRolloutStatus = getRollOutStatus

	TabHeader = " -"
)

type checker struct {
	context              context.Context
	out                  io.Writer
	client               *kubectl.CLI
	totalDeployments     int
	processedDeployments int32
}

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext, out io.Writer) error {
	client, err := pkgkubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting kubernetes client")
	}

	deadline := getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds)

	dMap, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller, deadline)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}

	wg := sync.WaitGroup{}
	// Its safe to use sync.Map without locks here as each subroutine adds a different key to the map.
	syncMap := &sync.Map{}

	checker := &checker{
		context:          ctx,
		out:              out,
		client:           kubectl.NewFromRunContext(runCtx),
		totalDeployments: len(dMap),
	}

	for dName, deadlineDuration := range dMap {
		wg.Add(1)
		go func(dName string, deadlineDuration time.Duration) {
			defer wg.Done()
			checker.pollDeploymentRolloutStatus(dName, deadlineDuration, syncMap)
		}(dName, deadlineDuration)
	}

	// Wait for all deployment status to be fetched
	wg.Wait()
	return getSkaffoldDeployStatus(syncMap)
}

func getDeployments(client kubernetes.Interface, ns string, l *DefaultLabeller, deadlineDuration time.Duration) (map[string]time.Duration, error) {
	deps, err := client.AppsV1().Deployments(ns).List(metav1.ListOptions{
		LabelSelector: l.RunIDKeyValueString(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch deployments")
	}

	depMap := map[string]time.Duration{}

	for _, d := range deps.Items {
		var deadline time.Duration
		if d.Spec.ProgressDeadlineSeconds == nil || *d.Spec.ProgressDeadlineSeconds > int32(deadlineDuration.Seconds()) {
			deadline = deadlineDuration
		} else {
			deadline = time.Duration(*d.Spec.ProgressDeadlineSeconds) * time.Second
		}
		depMap[d.Name] = deadline
	}

	return depMap, nil
}

func (c *checker) pollDeploymentRolloutStatus(dName string, deadline time.Duration, syncMap *sync.Map) {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(c.context, deadline+pollDuration)
	logrus.Debugf("checking rollout status %s", dName)
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			syncMap.Store(dName, errors.Wrap(timeoutContext.Err(), fmt.Sprintf("deployment rollout status could not be fetched within %v", deadline)))
			return
		case <-time.After(pollDuration):
			status, err := executeRolloutStatus(timeoutContext, c.client, dName)
			if err != nil || strings.Contains(status, "successfully rolled out") {
				syncMap.Store(dName, err)
				atomic.AddInt32(&c.processedDeployments, 1)
				c.printStatusCheckSummary(dName, err)
				return
			}
			syncMap.Store(dName, status)
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

func (c *checker) printStatusCheckSummary(dName string, err error) {
	status := fmt.Sprintf("%s deployment/%s", TabHeader, dName)
	if err != nil {
		status = fmt.Sprintf("%s failed.%s Error: %s.",
			status,
			trimNewLine(c.getPendingMessage()),
			trimNewLine(err.Error()),
		)
	} else {
		status = fmt.Sprintf("%s is ready.%s", status, c.getPendingMessage())
	}
	color.Default.Fprintln(c.out, status)
}

func (c *checker) getPendingMessage() string {
	if pendingDeployments := c.totalDeployments - int(c.processedDeployments); pendingDeployments > 0 {
		return fmt.Sprintf(" [%d/%d deployment(s) still pending]", pendingDeployments, c.totalDeployments)
	}
	return ""
}

func trimNewLine(msg string) string {
	return strings.TrimSuffix(msg, "\n")
}
