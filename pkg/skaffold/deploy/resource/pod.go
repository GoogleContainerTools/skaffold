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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/pkg/errors"
)

const (
	PodType     = "pod"
	podDeadline = 2 * time.Minute
)

type Pod struct {
	*Base
}

func NewPod(name string, ns string) *Pod {
	return &Pod{
		Base: &Base{name: name, namespace: ns, rType: PodType},
	}
}

func (p *Pod) CheckStatus(ctx context.Context, runCtx *runcontext.RunContext) {
	updated := newStatus("nyi", errors.New("not yet implemented"))
	if !p.status.Equal(updated) {
		p.status = updated
		p.done = true
	}
}

func (p *Pod) Deadline() time.Duration {
	return podDeadline
}
