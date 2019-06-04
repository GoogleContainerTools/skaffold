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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/pkg/errors"
)

var (
	deploymentOutputTemplate            = "{{range .items}}{{.metadata.name}}:{{.spec.progressDeadlineSeconds}}{{\",\"}}{{end}}"
	defaultStatusCheckDeadlineInSeconds = 600
)

func StatusCheck(ctx context.Context, out io.Writer, runCtx runcontext.RunContext) error {
	kubeCtl := kubectl.CLI{
		Namespace:   runCtx.Opts.Namespace,
		KubeContext: runCtx.KubeContext,
	}
	dMap, err := getDeployments(ctx, kubeCtl)
	for dName, deadline := range dMap {
		// Set the deadline to defaultStatusCheckDeadlineInSeconds if deadline is set to Math.MaxInt
		// See https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/extensions/v1beta1/defaults.go#L119
		if deadline == math.MaxInt32 {
			deadline = defaultStatusCheckDeadlineInSeconds
		}
		go fmt.Println(dName)
	}
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}

	return nil
}

func getDeployments(ctx context.Context, k kubectl.CLI) (map[string]int, error) {
	b, err := k.RunOut(ctx, nil, "get", []string{"deployments"}, "--output", fmt.Sprintf("go-template='%s'", deploymentOutputTemplate))
	if err != nil {
		return nil, err
	}
	m := map[string]int{}
	if len(b) == 0 {
		return m, nil
	}
	lines := strings.Split(string(b), ",")
	for _, line := range lines {
		kv := strings.Split(line, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("error parsing `kubectl get deployments` %s", line)
		}
		i, err := strconv.Atoi(kv[1])
		if err != nil {
			return m, err
		}

		m[kv[0]] = i
	}
	return m, nil

}
