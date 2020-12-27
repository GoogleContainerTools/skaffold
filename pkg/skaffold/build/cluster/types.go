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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Builder builds docker artifacts on Kubernetes.
type Builder struct {
	*latest.ClusterDetails

	cfg           Config
	kubectlcli    *kubectl.CLI
	mode          config.RunMode
	timeout       time.Duration
	artifactStore build.ArtifactStore
	teardownFunc  []func()
}

type Config interface {
	kubectl.Config
	docker.Config

	GetKubeContext() string
	Muted() config.Muted
	Mode() config.RunMode
}

// NewBuilder creates a new Builder that builds artifacts on cluster.
func NewBuilder(cfg Config, buildCfg *latest.ClusterDetails) (*Builder, error) {
	timeout, err := time.ParseDuration(buildCfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("parsing timeout: %w", err)
	}

	return &Builder{
		ClusterDetails: buildCfg,
		cfg:            cfg,
		kubectlcli:     kubectl.NewCLI(cfg, ""),
		mode:           cfg.Mode(),
		timeout:        timeout,
	}, nil
}

func (b *Builder) ArtifactStore(store build.ArtifactStore) {
	b.artifactStore = store
}

func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return nil
}
