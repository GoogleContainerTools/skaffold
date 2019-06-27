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
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/pkg/errors"
)

var (
	rolloutStatusTemplate       = "{{range .items}}{{.metadata.name}}:{{.spec.progressDeadlineSeconds}},{{end}}"
	// TODO: Move this to a flag or global config.
	defaultStatusCheckDeadlineInSeconds float32 = 10
	defaultPollPeriodInMilliseconds = 600
	// For testing
	executeRolloutStatus = getRollOutStatus
)

func StatusCheck(ctx context.Context, out io.Writer, runCtx *runcontext.RunContext) error {
	kubeCtl := kubectl.CLI{
		Namespace:   runCtx.Opts.Namespace,
		KubeContext: runCtx.KubeContext,
	}
	dMap, err := getDeadlineForDeployments(ctx, kubeCtl)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}
	w := sync.WaitGroup{}
	// Its safe to use sync.Map without locks here as each subroutine adds a different key.
	syncMap := &sync.Map{}

	for dName, deadline := range dMap {
		// Set the deadline to defaultStatusCheckDeadlineInSeconds if deadline is set to Math.MaxInt
		// See https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/extensions/v1beta1/defaults.go#L119
		if deadline == math.MaxInt32 {
			deadline = defaultStatusCheckDeadlineInSeconds
		}
		deadlineDuration := time.Duration(deadline) * time.Second
		w.Add(1)
		go func(dName string, deadlineDuration time.Duration) {
			defer w.Done()
			pollDeploymentsStatus(ctx, kubeCtl, dName, deadlineDuration, syncMap)
		}(dName, deadlineDuration)
	}

	// Wait for all deployment status to be fetched
	w.Wait()
	return getDeployStatus(syncMap, dMap)
}

func getDeadlineForDeployments(ctx context.Context, k kubectl.CLI) (map[string]float32, error) {
	skaffoldLabel := NewLabeller("").K8sManagedByLabelKeyValueString()
	b, err := k.RunOut(ctx, nil, "get", []string{"deployments"}, "-l", skaffoldLabel, "--output", fmt.Sprintf("go-template='%s'", rolloutStatusTemplate))
	if err != nil {
		return nil, err
	}
	deployments := map[string]float32{}
	if len(b) == 0 {
		return deployments, nil
	}

	lines := strings.Split(strings.Trim(string(b), "',"), ",")

	for _, line := range lines {
		kv := strings.Split(line, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("error parsing `kubectl get deployments` %s", line)
		}
		deadline, err := strconv.ParseFloat(kv[1], 32)
		if err != nil {
			return deployments, err
		}

		deployments[kv[0]] = float32(deadline)
	}
	return deployments, nil

}

func pollDeploymentsStatus(ctx context.Context, k kubectl.CLI, dName string, deadline time.Duration, syncMap *sync.Map) {
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
				syncMap.Store(dName, status)
				return
			}
		}
	}
}

func getDeployStatus(syncMap *sync.Map, deps map[string]float32) error {
	errorStrings := []string{}
	for d, _ := range deps {
		if errStr, ok := isErrorforValue(syncMap, d); ok {
			errorStrings = append(errorStrings, fmt.Sprintf("deployment %s failed due to %s", d, errStr))
		}
	}
	if len(errorStrings) == 0 {
		return nil
	}
	return fmt.Errorf("following deployments are not stable:\n%s", strings.Join(errorStrings, "\n"))
}

func getRollOutStatus(ctx context.Context, k kubectl.CLI, dName string) (string, error) {
	b, err := k.RunOut(ctx, nil, "rollout", []string{"status", "deployment", dName},
		"--watch=false")
	if err != nil {
		return "", err
	}
	return string(b), nil
}


func isErrorforValue(syncMap *sync.Map, d string) (string, bool) {
	v, ok := syncMap.Load(d)
	logrus.Debugf("rollout status for deployment %s is %v", d, v)
	if !ok {
		return "could not verify status for deployment", true
	}
	switch t := v.(type) {
	case error:
		return t.Error(), true
  default:
     return "", false
  }
}