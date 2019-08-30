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
	"fmt"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/pkg/errors"
)

const (
	PodType = "pod"
)

type Pod struct {
	*ResourceObj
}

func NewPod(name string, ns string) *Pod {
	return &Pod{
		ResourceObj: &ResourceObj{name: name, namespace: ns, rType: PodType},
	}
}

func (p *Pod) CheckStatus(ctx context.Context, runCtx *runcontext.RunContext) {
	client, err := kubernetesutil.GetClientset()
	if err != nil {
		p.UpdateStatus("", KubectlConnection, errors.Wrap(err, fmt.Sprintf("could not get status for %s", p.String())))
		return
	}
	if reason, err := kubernetesutil.GetPodDetails(client, p.Namespace(), p.Name()); err != nil {
		reason = parsePodDuplicateReasons(reason)
		p.UpdateStatus("", reason, err)
		return
	}
	p.UpdateStatus("", "",nil)
	p.checkComplete()
}

func (p *Pod) Deadline() time.Duration {
	return 2 * time.Minute
}

func (p *Pod) WithError(err error) *Pod {
	p.UpdateStatus("", "", err)
	return p
}

func parsePodDuplicateReasons(reason string) string {
	if reason == "ImgErrPull" || reason == "ImgPullBackOff" {
		return "ImgErrPull"
	}
	return reason
}
