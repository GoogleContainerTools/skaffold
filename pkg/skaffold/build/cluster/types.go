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

package cluster

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Builder builds docker artifacts on Kubernetes.
type Builder struct {
	*latest.ClusterDetails

	kubectlcli         *kubectl.CLI
	kubeContext        string
	timeout            time.Duration
	insecureRegistries map[string]bool
}

// NewBuilder creates a new Builder that builds artifacts on cluster.
func NewBuilder(runCtx *runcontext.RunContext) (*Builder, error) {
	timeout, err := time.ParseDuration(runCtx.Cfg.Build.Cluster.Timeout)
	if err != nil {
		return nil, fmt.Errorf("parsing timeout: %w", err)
	}

	return &Builder{
		ClusterDetails:     runCtx.Cfg.Build.Cluster,
		kubectlcli:         kubectl.NewFromRunContext(runCtx),
		timeout:            timeout,
		kubeContext:        runCtx.KubeContext,
		insecureRegistries: runCtx.InsecureRegistries,
	}, nil
}

func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return nil
}
