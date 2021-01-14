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

package sync

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type syncMap map[string][]string

type Item struct {
	Image    string
	Artifact string
	Copy     map[string][]string
	Delete   map[string][]string
}

type Syncer interface {
	Sync(context.Context, *Item) error
}

type podSyncer struct {
	kubectl    *pkgkubectl.CLI
	namespaces []string
}

type Config interface {
	kubectl.Config

	GetNamespaces() []string
	Deployers() []latest.DeployType
}

type SyncerMux struct {
	standaloneContainers []string
	cs                   *containerSyncer
	ps                   *podSyncer
}

func (m *SyncerMux) Sync(ctx context.Context, i *Item) error {
	if util.StrSliceContains(m.standaloneContainers, i.Artifact) {
		return m.cs.Sync(ctx, i)
	}
	return m.ps.Sync(ctx, i)
}

func NewSyncer(cfg Config) Syncer {
	mux := &SyncerMux{
		cs: &containerSyncer{},
		ps: &podSyncer{
			kubectl:    pkgkubectl.NewCLI(cfg, ""),
			namespaces: cfg.GetNamespaces(),
		},
	}

	for _, d := range cfg.Deployers() {
		if d.DockerDeploy != nil {
			mux.standaloneContainers = append(mux.standaloneContainers, d.DockerDeploy.Images...)
		}
	}

	return mux
}
