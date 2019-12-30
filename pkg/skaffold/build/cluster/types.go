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
	"io"
	"time"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Builder builds docker artifacts on Kubernetes.
type Builder struct {
	*latest.ClusterDetails

	docker      docker.DockerAPI
	kubectlcli  *kubectl.CLI
	kubeContext string
	timeout     time.Duration
}

// NewBuilder creates a new Builder that builds artifacts on cluster.
func NewBuilder(runCtx *runcontext.RunContext) (*Builder, error) {
	timeout, err := time.ParseDuration(runCtx.Cfg.Build.Cluster.Timeout)
	if err != nil {
		return nil, errors.Wrap(err, "parsing timeout")
	}

	return &Builder{
		ClusterDetails: runCtx.Cfg.Build.Cluster,
		kubectlcli:     kubectl.NewFromRunContext(runCtx),
		timeout:        timeout,
		kubeContext:    runCtx.KubeContext,
		docker:         docker.NewDockerAPI(runCtx),
	}, nil
}

// Labels are labels specific to cluster builder.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "cluster",
	}
}

func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return nil
}

func (b *Builder) SyncMap(ctx context.Context, artifact *latest.Artifact) (map[string][]string, error) {
	return nil, build.ErrSyncMapNotSupported{}
}
