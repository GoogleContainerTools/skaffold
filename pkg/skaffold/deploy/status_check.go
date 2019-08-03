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
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

var (
	defaultStatusCheckDeadline = time.Duration(10) * time.Minute

	// Poll period for checking set to 100 milliseconds
	defaultPollPeriodInMilliseconds = 100

	// For testing
	executeRolloutStatus = getRollOutStatus
)

func StatusCheck(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext) error {
	client, err := kubernetesutil.GetClientset()
	if err != nil {
		return err
	}
	deadline := getDeadline(runCtx.Cfg.Deploy.StatusCheckDeadlineSeconds)

	dMap, err := getDeployments(client, runCtx.Opts.Namespace, defaultLabeller, deadline)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}

	wg := sync.WaitGroup{}
	// Its safe to use sync.Map without locks here as each subroutine adds a different key to the map.
	syncMap := &sync.Map{}
	kubeCtl := kubectl.NewFromRunContext(runCtx)

	for dName, deadlineDuration := range dMap {
		wg.Add(1)
		go func(dName string, deadlineDuration time.Duration) {
			defer wg.Done()
			pollDeploymentRolloutStatus(ctx, kubeCtl, dName, deadlineDuration, syncMap)
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

func pollDeploymentRolloutStatus(ctx context.Context, k *kubectl.CLI, dName string, deadline time.Duration, syncMap *sync.Map) {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, deadline+pollDuration)
	logrus.Debugf("checking rollout status %s", dName)
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			syncMap.Store(dName, errors.Wrap(timeoutContext.Err(), fmt.Sprintf("deployment rollout status could not be fetched within %v", deadline)))
			return
		case <-time.After(pollDuration):
			status, err := executeRolloutStatus(timeoutContext, k, dName)
			if err != nil {
				syncMap.Store(dName, err)
				return
			}
			if strings.Contains(status, "successfully rolled out") {
				syncMap.Store(dName, nil)
				return
			}
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
