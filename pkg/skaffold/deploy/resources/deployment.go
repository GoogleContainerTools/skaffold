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

package resources

import (
	"context"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

const (
	DeploymentType    = "deployment"
	KubectlKilled     = "Killed"
	KubectlConnection = "KubectlConnection"
)

type Deployment struct {
	*ResourceObj
	deadline time.Duration
}

func NewDeployment(name string, ns string, deadline time.Duration) *Deployment {
	return &Deployment{
		ResourceObj: &ResourceObj{name: name, namespace: ns, rType: DeploymentType},
		deadline:    deadline,
	}
}

func (d *Deployment) CheckStatus(ctx context.Context, runCtx *runcontext.RunContext) {
	kubeCtl := kubectl.NewFromRunContext(runCtx)
	b, err := kubeCtl.RunOut(ctx, "rollout", "status", "deployment", d.name, "--namespace", d.namespace, "--watch=false")
	if err != nil {
		reason, details := parseKubectlError(err.Error())
		d.UpdateStatus(details, reason, err)
		if reason != KubectlConnection {
			d.checkComplete()
		}
		return
	}
	if s := string(b); strings.Contains(s, "successfully rolled out") {
		d.UpdateStatus(s, s, nil)
		d.checkComplete()
		return
	}
	d.UpdateStatus(string(b), string(b), nil)
}

func (d *Deployment) Deadline() time.Duration {
	return d.deadline
}

func parseKubectlError(errMsg string) (string, string) {
	errMsg = strings.TrimSuffix(errMsg, "\n")
	if strings.Contains(errMsg, "Unable to connect to the server") {
		return KubectlConnection, errMsg
	}
	if strings.Contains(errMsg, "signal: killed") {
		return KubectlKilled, "kubectl killed due to timeout"
	}
	return errMsg, errMsg
}

func (d *Deployment) WithError(err error) *Deployment {
	d.UpdateStatus("", err.Error(), err)
	return d
}

func (d *Deployment) WithStatus(status Status) *Deployment {
	d.status = status
	return d
}
