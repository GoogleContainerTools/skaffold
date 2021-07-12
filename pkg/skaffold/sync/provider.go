/*
Copyright 2021 The Skaffold Authors

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
	gosync "sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

type Provider interface {
	GetKubernetesSyncer(*kubernetes.ImageList, *kubectl.CLI) Syncer
	GetNoopSyncer() Syncer
}

type fullProvider struct {
	kubernetesSyncer func(*kubernetes.ImageList, *kubectl.CLI) Syncer
	noopSyncer       func() Syncer
}

var (
	provider *fullProvider
	once     gosync.Once
)

func NewSyncProvider(config Config) Provider {
	once.Do(func() {
		provider = &fullProvider{
			kubernetesSyncer: func(podSelector *kubernetes.ImageList, cli *kubectl.CLI) Syncer {
				return &podSyncer{
					kubectl: cli,
					config:  config,
				}
			},
			noopSyncer: func() Syncer {
				return nil
			},
		}
	})
	return provider
}

func (p *fullProvider) GetKubernetesSyncer(s *kubernetes.ImageList, cli *kubectl.CLI) Syncer {
	return p.kubernetesSyncer(s, cli)
}

func (p *fullProvider) GetNoopSyncer() Syncer {
	return p.noopSyncer()
}

type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesSyncer(*kubernetes.ImageList, *kubectl.CLI) Syncer {
	return &NoopSyncer{}
}

func (p *NoopProvider) GetNoopSyncer() Syncer {
	return &NoopSyncer{}
}
