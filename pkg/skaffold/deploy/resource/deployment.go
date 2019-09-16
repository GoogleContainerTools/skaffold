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

package resource

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

const (
	deploymentType = "deployment"
)

var (
	kubectlKilledErr     = errors.New("kubectl killed due to timeout")
	kubectlConnectionErr = errors.New("kubectl connection error")
)

type Deployment struct {
	*Base
	deadline time.Duration
}

func (d *Deployment) Deadline() time.Duration {
	return d.deadline
}

func NewDeployment(name string, ns string, deadline time.Duration) *Deployment {
	return &Deployment{
		Base: &Base{
			name:      name,
			namespace: ns,
			rType:     deploymentType,
			status:    newStatus("", nil),
		},
		deadline: deadline,
	}
}

func (d *Deployment) CheckStatus(ctx context.Context, runCtx *runcontext.RunContext) {
	kubeCtl := kubectl.NewFromRunContext(runCtx)
	b, err := kubeCtl.RunOut(ctx, "rollout", "status", "deployment", d.name, "--namespace", d.namespace, "--watch=false")
	details := string(b)
	err = parseKubectlError(err)
	d.UpdateStatus(details, err)
	if strings.Contains(details, "successfully rolled out") || isErrAndNotRetriable(err) {
		d.done = true
	}
}

func parseKubectlError(err error) error {
	if err == nil {
		return err
	}
	if strings.Contains(err.Error(), "Unable to connect to the server") {
		return kubectlConnectionErr
	}
	if strings.Contains(err.Error(), "signal: killed") {
		return kubectlKilledErr
	}
	return err
}

func isErrAndNotRetriable(err error) bool {
	if err == nil {
		return false
	}
	return err != kubectlConnectionErr
}

func (d *Deployment) MarkDone() {
	d.done = true
}
