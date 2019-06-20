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
	deploymentOutputTemplate = "{{range .items}}{{.metadata.name}}:{{.spec.progressDeadlineSeconds}}{{\",\"}}{{end}}"
	// TODO: Move this to a flag or global setting.
	defaultStatusCheckDeadlineInSeconds = 600
)

func StatusCheck(ctx context.Context, out io.Writer, runCtx runcontext.RunContext) error {
	kubeCtl := kubectl.CLI{
		Namespace:   runCtx.Opts.Namespace,
		KubeContext: runCtx.KubeContext,
	}
	dMap, err := getDeploymentsWithDeadline(ctx, kubeCtl)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}
	w := sync.WaitGroup{}
	syncMap := sync.Map{}

	for dName, deadline := range dMap {
		// Set the deadline to defaultStatusCheckDeadlineInSeconds if deadline is set to Math.MaxInt
		// See https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/extensions/v1beta1/defaults.go#L119
		if deadline == math.MaxInt32 {
			deadline = defaultStatusCheckDeadlineInSeconds
		}
		w.Add(1)
		fmt.Println(dName, deadline)
		go checkDeploymentsStatus(ctx , kubeCtl, dName, deadline, syncMap)
	}

	// Wait for all deployment status to be fetched
	w.Wait()
	return nil
	return getStatus(ctx, syncMap, dMap)
}

func getDeploymentsWithDeadline(ctx context.Context, k kubectl.CLI) (map[string]int, error) {
	b, err := k.RunOut(ctx, nil, "get", []string{"deployments"}, "--output", fmt.Sprintf("go-template='%s'", deploymentOutputTemplate))
	if err != nil {
		return nil, err
	}
	deployments := map[string]int{}
	if len(b) == 0 {
		return deployments, nil
	}
	lines := strings.Split(string(b), ",")
	for _, line := range lines {
		kv := strings.Split(line, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("error parsing `kubectl get deployments` %s", line)
		}
		deadline, err := strconv.Atoi(kv[1])
		if err != nil {
			return deployments, err
		}

		deployments[kv[0]] = deadline
	}
	return deployments, nil

}


func checkDeploymentsStatus(ctx context.Context, k kubectl.CLI, dName string, deadline int, syncMap sync.Map)  {
  timeoutContext, cancel := context.WithTimeout(ctx, time.Duration(deadline) * time.Second + 1)
  defer cancel()
	b, err := k.RunOut(timeoutContext, nil, "get", []string{"deployments"}, "--output", fmt.Sprintf("go-template='%s'", deploymentOutputTemplate))
	if err != nil {
		syncMap.Store(dName, b)
	}
	syncMap.Store(dName, err)
}

func getStatus(ctx context.Context, syncMap sync.Map, deps map[string]int) error  {
	errorStrings := []string{}
	for d, _ := range deps {
		v, ok  := syncMap.Load((d))
		if !ok {
			errorStrings = append(errorStrings, fmt.Sprintf("could not verify status for deployment %s", d))
		}
		switch t := v.(type) {
		case error:
			errorStrings = append(errorStrings, fmt.Sprintf("deployment %s failed due to %s", d, t.Error()))
		}
	}
	if len(errorStrings) == 0 {
		return nil
	}
	return fmt.Errorf("following deployments are not stable:\n%s", strings.Join(errorStrings, "\n"))
}